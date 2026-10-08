package session

import (
	"context"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/db"
	"github.com/stretchr/testify/require"
)

func currentProvider() (p config.ConfigProvider) {
	defer func() { _ = recover() }()
	return config.GetDefaultProvider()
}

func setupService(t *testing.T) (Service, context.Context) {
	t.Helper()
	old := currentProvider()
	dir := t.TempDir()
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{Data: config.Data{Directory: dir}}))
	t.Cleanup(func() { config.SetDefaultProvider(old) })

	sqlDB, err := db.Connect()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	return NewService(db.New(sqlDB)), context.Background()
}

func TestSessionCreateGetListSaveDelete(t *testing.T) {
	s, ctx := setupService(t)

	created, err := s.Create(ctx, "hello")
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, "hello", created.Title)

	got, err := s.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)

	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	got.Title = "renamed"
	got.PromptTokens = 10
	got.CompletionTokens = 20
	got.Cost = 0.5
	saved, err := s.Save(ctx, got)
	require.NoError(t, err)
	require.Equal(t, "renamed", saved.Title)
	require.Equal(t, int64(10), saved.PromptTokens)

	// summary message id round-trips through NullString
	got.SummaryMessageID = "m1"
	saved, err = s.Save(ctx, got)
	require.NoError(t, err)
	require.Equal(t, "m1", saved.SummaryMessageID)

	require.NoError(t, s.Delete(ctx, created.ID))
	_, err = s.Get(ctx, created.ID)
	require.Error(t, err)
	require.Error(t, s.Delete(ctx, "missing"))
}

func TestSessionTaskAndTitle(t *testing.T) {
	s, ctx := setupService(t)

	parent, err := s.Create(ctx, "parent")
	require.NoError(t, err)

	task, err := s.CreateTaskSession(ctx, "tool-1", parent.ID, "task")
	require.NoError(t, err)
	require.Equal(t, parent.ID, task.ParentSessionID)

	title, err := s.CreateTitleSession(ctx, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "title-"+parent.ID, title.ID)
	require.Equal(t, parent.ID, title.ParentSessionID)
}
