package tools

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/tools/shell"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/stretchr/testify/require"
)

func TestReadBashOutputGuidance(t *testing.T) {
	bootstrapConfig(t, t.TempDir())

	sh := shell.GetPersistentShell(config.WorkingDirectory())
	t.Cleanup(func() { sh.Close() })

	bashTool := NewBashTool(func() permission.Service {
		svc := permission.NewPermissionService()
		svc.AutoApproveSession("s")
		return svc
	}())

	// Explicit background: true.
	in, err := json.Marshal(BashParams{Command: "echo bg-output-check", Background: true})
	require.NoError(t, err)
	resp, err := bashTool.Run(testCtx("s", "m"), ToolCall{Input: string(in)})
	require.NoError(t, err)
	var startMeta BashResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &startMeta))
	require.NotEmpty(t, startMeta.ShellID)

	readTool := NewReadBashOutputTool(func() permission.Service {
		svc := permission.NewPermissionService()
		svc.AutoApproveSession("s")
		return svc
	}())

	// Poll while running; before completion it should be not finished.
	rin, err := json.Marshal(ReadBashOutputParams{ShellID: startMeta.ShellID})
	require.NoError(t, err)
	rr, err := readTool.Run(testCtx("s", "m"), ToolCall{Input: string(rin)})
	require.NoError(t, err)
	require.Contains(t, rr.Content, "still running")

	// After completion it should report Finished with the exit code and guidance.
	require.Eventually(t, func() bool {
		return sh.GetBackground(startMeta.ShellID).IsDone()
	}, 5*time.Second, 20*time.Millisecond)

	rr2, err := readTool.Run(testCtx("s", "m"), ToolCall{Input: string(rin)})
	require.NoError(t, err)
	require.Contains(t, rr2.Content, "finished")
	require.Contains(t, rr2.Content, "bg-output-check")

	var readMeta ReadBashOutputResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(rr2.Metadata), &readMeta))
	require.True(t, readMeta.Finished)
	require.Equal(t, 0, readMeta.ExitCode)

	// Unknown shell ID should error cleanly.
	bad, err := json.Marshal(ReadBashOutputParams{ShellID: "does-not-exist"})
	require.NoError(t, err)
	br, err := readTool.Run(testCtx("s", "m"), ToolCall{Input: string(bad)})
	require.NoError(t, err)
	require.True(t, br.IsError)
}
