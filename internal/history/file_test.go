package history

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
	ctx := context.Background()
	q := db.New(sqlDB)
	_, err = q.CreateSession(ctx, db.CreateSessionParams{ID: "s1", Title: "t"})
	require.NoError(t, err)
	return NewService(q, sqlDB), ctx
}

func TestHistoryCreateGetListDelete(t *testing.T) {
	s, ctx := setupService(t)

	f, err := s.Create(ctx, "s1", "a.txt", "hello")
	require.NoError(t, err)
	require.Equal(t, InitialVersion, f.Version)

	got, err := s.Get(ctx, f.ID)
	require.NoError(t, err)
	require.Equal(t, "hello", got.Content)

	got2, err := s.GetByPathAndSession(ctx, "a.txt", "s1")
	require.NoError(t, err)
	require.Equal(t, f.ID, got2.ID)

	list, err := s.ListBySession(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, list, 1)

	latest, err := s.ListLatestSessionFiles(ctx, "s1")
	require.NoError(t, err)
	require.Len(t, latest, 1)

	updated, err := s.Update(ctx, File{ID: f.ID, Content: "world", Version: "v1"})
	require.NoError(t, err)
	require.Equal(t, "world", updated.Content)

	require.NoError(t, s.Delete(ctx, f.ID))
	_, err = s.Get(ctx, f.ID)
	require.Error(t, err)
}

func TestHistoryVersions(t *testing.T) {
	s, ctx := setupService(t)

	_, err := s.CreateVersion(ctx, "s1", "b.txt", "v0")
	require.NoError(t, err)
	v1, err := s.CreateVersion(ctx, "s1", "b.txt", "v1-content")
	require.NoError(t, err)
	require.Equal(t, "v1", v1.Version)
	v2, err := s.CreateVersion(ctx, "s1", "b.txt", "v2-content")
	require.NoError(t, err)
	require.Equal(t, "v2", v2.Version)

	require.NoError(t, s.DeleteSessionFiles(ctx, "s1"))
	list, err := s.ListBySession(ctx, "s1")
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestHistoryMissing(t *testing.T) {
	s, ctx := setupService(t)
	_, err := s.Get(ctx, "nope")
	require.Error(t, err)
	require.Error(t, s.Delete(ctx, "nope"))
}
