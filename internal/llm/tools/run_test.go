package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/db"
	"github.com/kothagpt/kotha/internal/history"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/stretchr/testify/require"
)

func testCtx(sid, mid string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, SessionIDContextKey, sid)
	ctx = context.WithValue(ctx, MessageIDContextKey, mid)
	return ctx
}

func bootstrapConfig(t *testing.T, wd string) {
	t.Helper()
	old := func() config.ConfigProvider {
		defer func() { _ = recover() }()
		return config.GetDefaultProvider()
	}()
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{
		WorkingDir: wd,
		Data:       config.Data{Directory: t.TempDir()},
	}))
	t.Cleanup(func() { config.SetDefaultProvider(old) })
}

func TestBashValidation(t *testing.T) {
	bootstrapConfig(t, t.TempDir())
	b := NewBashTool(permission.NewPermissionService())

	// invalid JSON
	resp, err := b.Run(testCtx("s", "m"), ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// missing command
	in, _ := json.Marshal(BashParams{})
	resp, err = b.Run(testCtx("s", "m"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// banned command
	in, _ = json.Marshal(BashParams{Command: "curl http://x"})
	resp, err = b.Run(testCtx("s", "m"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "not allowed")

	// missing ctx values
	in, _ = json.Marshal(BashParams{Command: "echo hi"})
	_, err = b.Run(context.Background(), ToolCall{Input: string(in)})
	require.Error(t, err)
}

func TestBashSafeReadOnly(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	b := NewBashTool(permission.NewPermissionService())
	in, _ := json.Marshal(BashParams{Command: "echo hello"})
	resp, err := b.Run(testCtx("s", "m"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content, "hello")
}

type denyPermission struct{ permission.Service }

func (denyPermission) Request(permission.CreatePermissionRequest) bool { return false }

func TestBashPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	b := NewBashTool(denyPermission{permission.NewPermissionService()})
	in, _ := json.Marshal(BashParams{Command: "mkdir denied-dir-xyz"})
	_, err := b.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.ErrorIs(t, err, permission.ErrorPermissionDenied)

	// allow path executes
	svc2 := permission.NewPermissionService()
	svc2.AutoApproveSession("s1")
	b2 := NewBashTool(svc2)
	in2, _ := json.Marshal(BashParams{Command: "mkdir " + filepath.Join(dir, "allowed-dir-xyz")})
	resp, err := b2.Run(testCtx("s1", "m1"), ToolCall{Input: string(in2)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.DirExists(t, filepath.Join(dir, "allowed-dir-xyz"))
}

func TestTruncateHelpers(t *testing.T) {
	require.Equal(t, "abc", truncateOutput("abc"))
	require.Equal(t, 0, countLines(""))
	require.Equal(t, 3, countLines("a\nb\nc"))

	long := string(make([]byte, MaxOutputLength+100))
	for i := range []byte(long) {
		long = long[:i] + "x" + long[i+1:]
	}
	out := truncateOutput(long)
	require.Contains(t, out, "truncated")
}

func TestWriteValidation(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	svc := permission.NewPermissionService()
	svc.AutoApproveSession("s1")
	sqlDB, err := db.Connect()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()
	q := db.New(sqlDB)
	_, err = q.CreateSession(context.Background(), db.CreateSessionParams{ID: "s1", Title: "t"})
	require.NoError(t, err)
	h := history.NewService(q, sqlDB)

	w := NewWriteTool(nil, svc, h)

	// bad JSON
	resp, err := w.Run(testCtx("s1", "m1"), ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// missing fields
	in, _ := json.Marshal(WriteParams{})
	resp, err = w.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// directory path
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))
	in, _ = json.Marshal(WriteParams{FilePath: "subdir", Content: "x"})
	resp, err = w.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// success: new file
	in, _ = json.Marshal(WriteParams{FilePath: "new.txt", Content: "hello"})
	resp, err = w.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	data, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))

	// idempotent: same content errors
	resp, err = w.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "already contains")

	// update existing
	in, _ = json.Marshal(WriteParams{FilePath: "new.txt", Content: "hello2"})
	recordFileRead(filepath.Join(dir, "new.txt"))
	resp, err = w.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	// missing ctx
	inMissing, _ := json.Marshal(WriteParams{FilePath: "new.txt", Content: "hello3"})
	_, err = w.Run(context.Background(), ToolCall{Input: string(inMissing)})
	require.Error(t, err)
}

func TestWritePermissionDenied(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	sqlDB, err := db.Connect()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()
	q := db.New(sqlDB)
	_, err = q.CreateSession(context.Background(), db.CreateSessionParams{ID: "s9", Title: "t"})
	require.NoError(t, err)
	h := history.NewService(q, sqlDB)
	w := NewBashTool(denyPermission{permission.NewPermissionService()})
	_ = w
	w2 := NewWriteTool(nil, denyPermission{permission.NewPermissionService()}, h)

	in, _ := json.Marshal(WriteParams{FilePath: "denied.txt", Content: "x"})
	_, err = w2.Run(testCtx("s9", "m9"), ToolCall{Input: string(in)})
	require.ErrorIs(t, err, permission.ErrorPermissionDenied)
	_ = dir
}
