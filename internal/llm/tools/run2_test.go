package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kothagpt/kotha/internal/db"
	"github.com/kothagpt/kotha/internal/history"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/stretchr/testify/require"
)

func editFixture(t *testing.T, sid string) (BaseTool, string) {
	t.Helper()
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	svc := permission.NewPermissionService()
	svc.AutoApproveSession(sid)
	sqlDB, err := db.Connect()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	q := db.New(sqlDB)
	_, err = q.CreateSession(context.Background(), db.CreateSessionParams{ID: sid, Title: "t"})
	require.NoError(t, err)
	h := history.NewService(q, sqlDB)
	return NewEditTool(nil, svc, h), dir
}

func TestEditValidation(t *testing.T) {
	e, _ := editFixture(t, "s1")
	resp, err := e.Run(testCtx("s1", "m1"), ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ := json.Marshal(EditParams{})
	resp, err = e.Run(testCtx("s1", "m1"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestEditCreateReplaceDelete(t *testing.T) {
	e, dir := editFixture(t, "s2")
	ctx := testCtx("s2", "m2")
	fp := filepath.Join(dir, "f.txt")

	// create
	in, _ := json.Marshal(EditParams{FilePath: fp, NewString: "line1\nline2\nline3\n"})
	resp, err := e.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	data, err := os.ReadFile(fp)
	require.NoError(t, err)
	require.Contains(t, string(data), "line1")

	// replace
	recordFileRead(fp)
	in, _ = json.Marshal(EditParams{FilePath: fp, OldString: "line2", NewString: "LINE2"})
	resp, err = e.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	data, _ = os.ReadFile(fp)
	require.Contains(t, string(data), "LINE2")

	// not found
	recordFileRead(fp)
	in, _ = json.Marshal(EditParams{FilePath: fp, OldString: "zzz-nope", NewString: "x"})
	resp, err = e.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// delete content
	recordFileRead(fp)
	in, _ = json.Marshal(EditParams{FilePath: fp, OldString: "LINE2\n"})
	resp, err = e.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	data, _ = os.ReadFile(fp)
	require.NotContains(t, string(data), "LINE2")
}

func TestEditDuplicateMatch(t *testing.T) {
	e, dir := editFixture(t, "s3")
	ctx := testCtx("s3", "m3")
	fp := filepath.Join(dir, "d.txt")
	require.NoError(t, os.WriteFile(fp, []byte("aa\nbb\naa\n"), 0o644))
	recordFileRead(fp)
	in, _ := json.Marshal(EditParams{FilePath: fp, OldString: "aa", NewString: "x"})
	resp, err := e.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "multiple times")
}

func TestViewGrepGlobLsRun(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\nworld\n"), 0o644))
	ctx := testCtx("s", "m")

	v := NewViewTool(nil)
	in, _ := json.Marshal(ViewParams{FilePath: filepath.Join(dir, "a.txt")})
	resp, err := v.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content, "hello")

	in, _ = json.Marshal(ViewParams{FilePath: filepath.Join(dir, "missing.txt")})
	resp, err = v.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	g := NewGrepTool()
	gin, _ := json.Marshal(GrepParams{Pattern: "hello", Path: dir})
	resp, err = g.Run(ctx, ToolCall{Input: string(gin)})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	gl := NewGlobTool()
	gin, _ = json.Marshal(GlobParams{Pattern: "*.txt", Path: dir})
	resp, err = gl.Run(ctx, ToolCall{Input: string(gin)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content, "a.txt")

	l := NewLsTool()
	lin, _ := json.Marshal(LSParams{Path: dir})
	resp, err = l.Run(ctx, ToolCall{Input: string(lin)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestPatchValidationAndAdd(t *testing.T) {
	dir := t.TempDir()
	bootstrapConfig(t, dir)
	sid := "sp"
	svc := permission.NewPermissionService()
	svc.AutoApproveSession(sid)
	sqlDB, err := db.Connect()
	require.NoError(t, err)
	defer sqlDB.Close()
	q := db.New(sqlDB)
	_, err = q.CreateSession(context.Background(), db.CreateSessionParams{ID: sid, Title: "t"})
	require.NoError(t, err)
	h := history.NewService(q, sqlDB)
	p := NewPatchTool(nil, svc, h)
	ctx := testCtx(sid, "m")

	// bad JSON / empty
	resp, err := p.Run(ctx, ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	in, _ := json.Marshal(PatchParams{})
	resp, err = p.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	// add-file patch
	text := "*** Begin Patch\n*** Add File: added.txt\n+hello\n+world\n*** End Patch"
	in, _ = json.Marshal(PatchParams{PatchText: text})
	resp, err = p.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	data, err := os.ReadFile(filepath.Join(dir, "added.txt"))
	require.NoError(t, err)
	require.Contains(t, string(data), "hello")
}
