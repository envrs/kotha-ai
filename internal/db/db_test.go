package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDatabaseCRUD(t *testing.T) {
	q, _ := OpenTestDB(t)
	ctx := context.Background()
	s1, err := q.CreateSession(ctx, CreateSessionParams{
		ID:    "sess-1",
		Title: "first",
		Cost:  0.5,
	})
	require.NoError(t, err)
	require.Equal(t, "sess-1", s1.ID)
	require.Equal(t, "first", s1.Title)
	require.Equal(t, 0.5, s1.Cost)
	require.NotZero(t, s1.CreatedAt)

	_, err = q.CreateSession(ctx, CreateSessionParams{
		ID:              "sess-2",
		Title:           "child",
		ParentSessionID: sql.NullString{String: "sess-1", Valid: true},
	})
	require.NoError(t, err)

	got, err := q.GetSessionByID(ctx, "sess-1")
	require.NoError(t, err)
	require.True(t, got.ParentSessionID.Valid == false)

	// ListSessions returns root sessions only; sess-2 is a child.
	sessions, err := q.ListSessions(ctx)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "sess-1", sessions[0].ID)

	updated, err := q.UpdateSession(ctx, UpdateSessionParams{
		Title:            "renamed",
		PromptTokens:     10,
		CompletionTokens: 5,
		Cost:             1.25,
		ID:               "sess-1",
	})
	require.NoError(t, err)
	require.Equal(t, "renamed", updated.Title)
	require.Equal(t, int64(10), updated.PromptTokens)
	require.Equal(t, 1.25, updated.Cost)

	// ---- messages ----
	m1, err := q.CreateMessage(ctx, CreateMessageParams{
		ID:        "msg-1",
		SessionID: "sess-1",
		Role:      "user",
		Parts:     `[{"text":"hello"}]`,
	})
	require.NoError(t, err)
	require.Equal(t, "user", m1.Role)

	_, err = q.CreateMessage(ctx, CreateMessageParams{
		ID:        "msg-2",
		SessionID: "sess-1",
		Role:      "assistant",
		Parts:     `[{"text":"hi"}]`,
	})
	require.NoError(t, err)

	msgs, err := q.ListMessagesBySession(ctx, "sess-1")
	require.NoError(t, err)
	require.Len(t, msgs, 2)

	gotMsg, err := q.GetMessage(ctx, "msg-1")
	require.NoError(t, err)
	require.Equal(t, "msg-1", gotMsg.ID)
	require.Contains(t, gotMsg.Parts, "hello")

	require.NoError(t, q.UpdateMessage(ctx, UpdateMessageParams{
		Parts:      `[{"text":"updated"}]`,
		FinishedAt: sql.NullInt64{Int64: 12345, Valid: true},
		ID:         "msg-1",
	}))
	gotMsg, err = q.GetMessage(ctx, "msg-1")
	require.NoError(t, err)
	require.Contains(t, gotMsg.Parts, "updated")
	require.Equal(t, int64(12345), gotMsg.FinishedAt.Int64)

	// ---- files ----
	_, err = q.CreateFile(ctx, CreateFileParams{
		ID:        "file-1",
		SessionID: "sess-1",
		Path:      "/a.txt",
		Content:   "alpha",
		Version:   "1",
	})
	require.NoError(t, err)
	_, err = q.CreateFile(ctx, CreateFileParams{
		ID:        "file-2",
		SessionID: "sess-1",
		Path:      "/b.txt",
		Content:   "beta",
		Version:   "1",
	})
	require.NoError(t, err)
	_, err = q.CreateFile(ctx, CreateFileParams{
		ID:        "file-3",
		SessionID: "sess-2",
		Path:      "/new.txt",
		Content:   "new",
		Version:   "1",
	})
	require.NoError(t, err)

	f, err := q.GetFile(ctx, "file-1")
	require.NoError(t, err)
	require.Equal(t, "alpha", f.Content)

	byPath, err := q.GetFileByPathAndSession(ctx, GetFileByPathAndSessionParams{
		Path:      "/a.txt",
		SessionID: "sess-1",
	})
	require.NoError(t, err)
	require.Equal(t, "file-1", byPath.ID)

	sessFiles, err := q.ListFilesBySession(ctx, "sess-1")
	require.NoError(t, err)
	require.Len(t, sessFiles, 2)

	pathFiles, err := q.ListFilesByPath(ctx, "/a.txt")
	require.NoError(t, err)
	require.Len(t, pathFiles, 1)

	latest, err := q.ListLatestSessionFiles(ctx, "sess-1")
	require.NoError(t, err)
	require.Len(t, latest, 2)

	newFiles, err := q.ListNewFiles(ctx)
	require.NoError(t, err)
	require.Len(t, newFiles, 3)
	require.Contains(t, []string{newFiles[0].Path, newFiles[1].Path, newFiles[2].Path}, "/new.txt")

	updatedFile, err := q.UpdateFile(ctx, UpdateFileParams{
		Content: "alpha2",
		Version: "2",
		ID:      "file-1",
	})
	require.NoError(t, err)
	require.Equal(t, "alpha2", updatedFile.Content)
	require.Equal(t, "2", updatedFile.Version)

	// ---- deletes ----
	require.NoError(t, q.DeleteFile(ctx, "file-2"))
	_, err = q.GetFile(ctx, "file-2")
	require.Error(t, err)

	require.NoError(t, q.DeleteMessage(ctx, "msg-2"))
	_, err = q.GetMessage(ctx, "msg-2")
	require.Error(t, err)

	require.NoError(t, q.DeleteSessionMessages(ctx, "sess-1"))
	msgs, err = q.ListMessagesBySession(ctx, "sess-1")
	require.NoError(t, err)
	require.Len(t, msgs, 0)

	require.NoError(t, q.DeleteSessionFiles(ctx, "sess-2"))
	require.NoError(t, q.DeleteSession(ctx, "sess-2"))
	_, err = q.GetSessionByID(ctx, "sess-2")
	require.Error(t, err)

	sessions, err = q.ListSessions(ctx)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
}

func TestConnectRequiresDataDir(t *testing.T) {
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{}))
	_, err := Connect()
	require.Error(t, err)
	require.Contains(t, err.Error(), "data.dir is not set")
}
