package agent

import (
	"context"

	"github.com/kothagpt/kotha/ai/internal/config"
	"github.com/kothagpt/kotha/ai/internal/history"
	"github.com/kothagpt/kotha/ai/internal/llm/tools"
	"github.com/kothagpt/kotha/ai/internal/logging"
	"github.com/kothagpt/kotha/ai/internal/lsp"
	"github.com/kothagpt/kotha/ai/internal/message"
	"github.com/kothagpt/kotha/ai/internal/permission"
	"github.com/kothagpt/kotha/ai/internal/session"
)

func CoderAgentTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	lspClients map[string]*lsp.Client,
	configProvider config.ConfigProvider,
	logger logging.Logger,
) []tools.BaseTool {
	ctx := context.Background()
	otherTools := GetMcpTools(ctx, permissions, configProvider, logger)
	if len(lspClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(lspClients))
	}
	return append(
		[]tools.BaseTool{
			tools.NewBashTool(permissions),
			tools.NewEditTool(lspClients, permissions, history),
			tools.NewFetchTool(permissions),
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewLsTool(),
			tools.NewSourcegraphTool(),
			tools.NewViewTool(lspClients),
			tools.NewPatchTool(lspClients, permissions, history),
			tools.NewWriteTool(lspClients, permissions, history),
			NewAgentTool(sessions, messages, lspClients, configProvider, logger),
		}, otherTools...,
	)
}

func TaskAgentTools(lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(lspClients),
	}
}
