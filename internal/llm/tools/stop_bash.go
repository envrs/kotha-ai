package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/tools/shell"
	"github.com/kothagpt/kotha/internal/permission"
)

type StopBashParams struct {
	ShellID string `json:"shell_id"`
}

type StopBashResponseMetadata struct {
	ShellID string `json:"shell_id"`
}

const (
	StopBashToolName = "stop_bash"
	stopBashDescription = `Stops a background bash command.

WHEN TO USE THIS TOOL:
- Use when you need to stop a background bash command that is no longer needed
- Use when a background command is taking too long or producing unexpected output

Before using this tool:
1. Verify that the shell ID is correct and the command is still running

Parameters:
- shell_id: The ID of the background bash command to stop

Usage notes:
- The command will be killed and its children will be terminated
- After stopping, the output file will contain any output produced before the command was stopped
`
)

type stopBashTool struct {
	permissions permission.Service
}

func NewStopBashTool(permission permission.Service) BaseTool {
	return &stopBashTool{
		permissions: permission,
	}
}

func (s *stopBashTool) Info() ToolInfo {
	return ToolInfo{
		Name:        StopBashToolName,
		Description: stopBashDescription,
		Parameters: map[string]any{
			"shell_id": map[string]any{
				"type":        "string",
				"description": "The ID of the background bash command to stop",
			},
		},
		Required: []string{"shell_id"},
	}
}

func (s *stopBashTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params StopBashParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("invalid parameters"), nil
	}
	if params.ShellID == "" {
		return NewTextErrorResponse("missing shell_id"), nil
	}

	shell := shell.GetPersistentShell(config.WorkingDirectory())
	if !shell.CancelBackground(params.ShellID) {
		return NewTextErrorResponse(fmt.Sprintf("background command not found: %s", params.ShellID)), nil
	}

	metadata := StopBashResponseMetadata(params)
	return WithResponseMetadata(NewTextResponse(fmt.Sprintf("Background command %s stopped", params.ShellID)), metadata), nil
}