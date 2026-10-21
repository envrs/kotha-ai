package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/tools/shell"
	"github.com/kothagpt/kotha/internal/permission"
)

type ReadBashOutputParams struct {
	ShellID string `json:"shell_id"`
}

type ReadBashOutputPermissionsParams struct {
	ShellID string `json:"shell_id"`
}

type ReadBashOutputResponseMetadata struct {
	ShellID    string `json:"shell_id"`
	OutputFile string `json:"output_file"`
	ErrorFile  string `json:"error_file"`
	Finished   bool   `json:"finished"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

const (
	ReadBashOutputToolName = "read_bash_output"
	readBashOutputDescription = `Read the output of a background bash command.

WHEN TO USE THIS TOOL:
- Use when you need to check the output of a background command that was started with the Bash tool's background option
- Use to read interim output while a long-running command is still executing

Before using this tool:
1. Verify that the shell ID is correct and the command is still running

Parameters:
- shell_id: The ID of the background bash command

Usage notes:
- The output file is updated in real-time as the command produces output
- If the command has finished, the exit code is included in the response
- If the command is still running, you can poll this tool periodically
`
)

type readBashOutputTool struct {
	permissions permission.Service
}

func NewReadBashOutputTool(permission permission.Service) BaseTool {
	return &readBashOutputTool{
		permissions: permission,
	}
}

func (r *readBashOutputTool) Info() ToolInfo {
	return ToolInfo{
		Name:        ReadBashOutputToolName,
		Description: readBashOutputDescription,
		Parameters: map[string]any{
			"shell_id": map[string]any{
				"type":        "string",
				"description": "The ID of the background bash command",
			},
		},
		Required: []string{"shell_id"},
	}
}

func (r *readBashOutputTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params ReadBashOutputParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}
	if params.ShellID == "" {
		return NewTextErrorResponse("missing shell_id"), nil
	}

	shellInst := shell.GetPersistentShell(config.WorkingDirectory())
	bc := shellInst.GetBackground(params.ShellID)
	if bc == nil {
		return NewTextErrorResponse(fmt.Sprintf("background command not found: %s", params.ShellID)), nil
	}

	// Read the live output file.
	stdout := shell.ReadFileOrEmpty(bc.OutputFile)
	stderr := shell.ReadFileOrEmpty(bc.ErrorFile)

	metadata := ReadBashOutputResponseMetadata{
		ShellID:    bc.ID,
		OutputFile: bc.OutputFile,
		ErrorFile:  bc.ErrorFile,
		Finished:   bc.IsDone(),
	}

	if bc.IsDone() {
		res := bc.Result()
		if res != nil {
			metadata.ExitCode = res.ExitCode
		}
	}

	output := stdout
	if stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr
	}
	if output == "" {
		output = "(no output yet)"
	}

	if bc.IsDone() {
		res := bc.Result()
		if res != nil && res.Err != nil {
			output += fmt.Sprintf("\nError: %v", res.Err)
		}
		if res != nil && res.ExitCode != 0 {
			output += fmt.Sprintf("\nCommand finished with exit code %d.", res.ExitCode)
		} else {
			output += "\nCommand finished successfully (exit code 0)."
		}
	} else {
		output += "\nCommand is still running. Poll again with this tool to receive more output."
	}

	return WithResponseMetadata(NewTextResponse(output), metadata), nil
}