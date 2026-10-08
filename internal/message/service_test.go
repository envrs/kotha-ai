package message

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/db"
	"github.com/stretchr/testify/require"
)

type fakeQuerier struct {
	db.Querier
	msgs map[string]db.Message
}

func newFake() *fakeQuerier { return &fakeQuerier{msgs: map[string]db.Message{}} }

func (f *fakeQuerier) CreateMessage(_ context.Context, arg db.CreateMessageParams) (db.Message, error) {
	m := db.Message{ID: arg.ID, SessionID: arg.SessionID, Role: arg.Role, Parts: arg.Parts, Model: arg.Model}
	f.msgs[m.ID] = m
	return m, nil
}

func (f *fakeQuerier) GetMessage(_ context.Context, id string) (db.Message, error) {
	m, ok := f.msgs[id]
	if !ok {
		return db.Message{}, errors.New("not found")
	}
	return m, nil
}

func (f *fakeQuerier) ListMessagesBySession(_ context.Context, sid string) ([]db.Message, error) {
	var out []db.Message
	for _, m := range f.msgs {
		if m.SessionID == sid {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeQuerier) UpdateMessage(_ context.Context, arg db.UpdateMessageParams) error {
	m, ok := f.msgs[arg.ID]
	if !ok {
		return errors.New("not found")
	}
	m.Parts = arg.Parts
	f.msgs[arg.ID] = m
	return nil
}

func (f *fakeQuerier) DeleteMessage(_ context.Context, id string) error {
	if _, ok := f.msgs[id]; !ok {
		return errors.New("not found")
	}
	delete(f.msgs, id)
	return nil
}

func TestServiceCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewService(newFake())

	// user role gets auto Finish part
	m, err := s.Create(ctx, "s1", CreateMessageParams{Role: User, Parts: []ContentPart{TextContent{Text: "hi"}}})
	require.NoError(t, err)
	require.True(t, m.IsFinished())

	got, err := s.Get(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, "hi", got.Content().Text)

	list, err := s.List(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, list, 1)

	m.Parts = append(m.Parts, TextContent{Text: "more"})
	require.NoError(t, s.Update(ctx, m))
	got, _ = s.Get(ctx, m.ID)
	require.Len(t, got.Parts, 3)

	require.NoError(t, s.Delete(ctx, m.ID))
	_, err = s.Get(ctx, m.ID)
	require.Error(t, err)
	require.Error(t, s.Delete(ctx, "missing"))
}

func TestServiceAssistantNoAutoFinish(t *testing.T) {
	ctx := context.Background()
	s := NewService(newFake())
	m, err := s.Create(ctx, "s1", CreateMessageParams{Role: Assistant, Parts: []ContentPart{TextContent{Text: "a"}}})
	require.NoError(t, err)
	require.False(t, m.IsFinished())
	require.NoError(t, s.DeleteSessionMessages(ctx, "s1"))
	list, err := s.List(ctx, "s1")
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestServiceUnknownPartErrors(t *testing.T) {
	ctx := context.Background()
	s := NewService(newFake())
	_, err := s.Create(ctx, "s1", CreateMessageParams{Role: User, Parts: []ContentPart{badPart{}}})
	require.Error(t, err)
	require.Error(t, s.Update(ctx, Message{ID: "x", Parts: []ContentPart{badPart{}}}))
}

func TestNullModelRoundtrip(t *testing.T) {
	ctx := context.Background()
	f := newFake()
	f.msgs["m1"] = db.Message{ID: "m1", SessionID: "s", Role: "user", Parts: `[]`, Model: sql.NullString{}}
	s := NewService(f)
	got, err := s.Get(ctx, "m1")
	require.NoError(t, err)
	require.Empty(t, string(got.Model))
}
