package symbol

import (
	"context"
	"testing"

	"github.com/kothagpt/kotha/internal/db"
	"github.com/stretchr/testify/require"
)

func openSymbolService(t *testing.T) Service {
	t.Helper()
	q, conn := db.OpenTestDB(t)
	t.Cleanup(func() { _ = conn.Close() })
	ctx := context.Background()
	_, err := q.CreateSession(ctx, db.CreateSessionParams{ID: "sess-1", Title: "test"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return NewService(q)
}

func TestSymbolCreateGetListSearchDelete(t *testing.T) {
	svc := openSymbolService(t)
	ctx := context.Background()

	sym, err := svc.Create(ctx, Symbol{
		SessionID: "sess-1",
		FilePath:  "pkg/foo.go",
		Name:      "MyFunc",
		Kind:      "12",
		Range:    "0:0-5:10",
		Language: "go",
	})
	require.NoError(t, err)
	require.NotEmpty(t, sym.ID)
	require.Equal(t, "MyFunc", sym.Name)

	got, err := svc.Get(ctx, sym.ID)
	require.NoError(t, err)
	require.Equal(t, sym.ID, got.ID)

	bySession, err := svc.ListBySession(ctx, "sess-1")
	require.NoError(t, err)
	require.Len(t, bySession, 1)

	byFile, err := svc.ListByFile(ctx, "pkg/foo.go", "sess-1")
	require.NoError(t, err)
	require.Len(t, byFile, 1)

	results, err := svc.Search(ctx, "sess-1", "myf")
	require.NoError(t, err)
	require.Len(t, results, 1)

	empty, err := svc.Search(ctx, "sess-1", "zzz")
	require.NoError(t, err)
	require.Len(t, empty, 0)

	err = svc.Delete(ctx, sym.ID)
	require.NoError(t, err)

	_, err = svc.Get(ctx, sym.ID)
	require.Error(t, err)
}

func TestSymbolDeleteFileSymbols(t *testing.T) {
	svc := openSymbolService(t)
	ctx := context.Background()

	sym, err := svc.Create(ctx, Symbol{
		SessionID: "sess-1",
		FilePath:  "pkg/foo.go",
		Name:      "MyFunc",
		Kind:      "12",
		Language:  "go",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteFileSymbols(ctx, "pkg/foo.go", "sess-1"))

	_, err = svc.Get(ctx, sym.ID)
	require.Error(t, err)
}

func TestSymbolDeleteSessionSymbols(t *testing.T) {
	svc := openSymbolService(t)
	ctx := context.Background()

	sym, err := svc.Create(ctx, Symbol{
		SessionID: "sess-1",
		FilePath:  "pkg/foo.go",
		Name:      "MyFunc",
		Kind:      "12",
		Language:  "go",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteSessionSymbols(ctx, "sess-1"))

	_, err = svc.Get(ctx, sym.ID)
	require.Error(t, err)
}