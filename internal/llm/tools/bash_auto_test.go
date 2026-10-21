package tools

import (
	"encoding/json"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/tools/shell"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/stretchr/testify/require"
)

func TestBashToolAutoBackgroundOnTimeout(t *testing.T) {
	bootstrapConfig(t, t.TempDir())

	// Close the persistent shell when the test ends so no long-running
	// background process keeps the test binary's stdout open.
	sh := shell.GetPersistentShell(config.WorkingDirectory())
	t.Cleanup(func() { sh.Close() })

	// Command sleeps longer than its timeout, so it gets interrupted and
	// auto-backgrounded.
	svc := permission.NewPermissionService()
	svc.AutoApproveSession("s")
	tool := NewBashTool(svc)
	input, err := json.Marshal(BashParams{
		Command: "sleep 10",
		Timeout: 500, // 500ms
	})
	require.NoError(t, err)

	resp, err := tool.Run(testCtx("s", "m"), ToolCall{
		Name:  BashToolName,
		Input: string(input),
	})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "moved to the background")
	require.Contains(t, resp.Content, "Shell ID")
	require.Contains(t, resp.Content, "Output file")
	require.Contains(t, resp.Content, "Error file")
	require.Contains(t, resp.Content, "read_bash_output")

	// The metadata should carry the shell ID for downstream polling.
	var meta BashResponseMetadata
	require.NoError(t, json.Unmarshal([]byte(resp.Metadata), &meta))
	require.NotEmpty(t, meta.ShellID)
	require.NotEmpty(t, meta.OutputFile)
	require.NotEmpty(t, meta.ErrorFile)
}
