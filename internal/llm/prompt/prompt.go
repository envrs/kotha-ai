package prompt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kothagpt/kotha/internal/concurrency"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/logging"
)

func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	case config.AgentCoder:
		basePrompt = CoderPrompt(provider)
	case config.AgentTitle:
		basePrompt = TitlePrompt(provider)
	case config.AgentTask:
		basePrompt = TaskPrompt(provider)
	case config.AgentSummarizer:
		basePrompt = SummarizerPrompt(provider)
	default:
		basePrompt = "You are a helpful assistant"
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		// Add context from project-specific instruction files if they exist
		contextContent := getContextFromPaths()
		logging.Debug("Context content", "Context", contextContent)
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\n Make sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

var (
	onceContext    sync.Once
	contextContent string
)

func getContextFromPaths() string {
	onceContext.Do(func() {
		var (
			cfg          = config.Get()
			workDir      = cfg.WorkingDir
			contextPaths = cfg.ContextPaths
		)

		contextContent = processContextPaths(workDir, contextPaths)
	})

	return contextContent
}

func processContextPaths(workDir string, paths []string) string {
	ctx := context.Background()
	// Bounded fan-out: one worker per path, max 8 concurrent file walks.
	// Results collected under mutex; no result channel to close/leak.
	var (
		mu      sync.Mutex
		ordered = make([][]string, len(paths))
	)
	processed := make(map[string]bool)

	gr := concurrency.WithGroup(ctx, 8)
	for idx, path := range paths {
		gr.Go(func(ctx context.Context) error {
			var local []string
			if strings.HasSuffix(path, "/") {
				_ = filepath.WalkDir(filepath.Join(workDir, path), func(fpath string, d os.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					}
					mu.Lock()
					lower := strings.ToLower(fpath)
					dup := processed[lower]
					if !dup {
						processed[lower] = true
					}
					mu.Unlock()
					if dup {
						return nil
					}
					if result := processFile(fpath); result != "" {
						local = append(local, result)
					}
					return nil
				})
			} else {
				fullPath := filepath.Join(workDir, path)
				mu.Lock()
				lower := strings.ToLower(fullPath)
				dup := processed[lower]
				if !dup {
					processed[lower] = true
				}
				mu.Unlock()
				if !dup {
					if result := processFile(fullPath); result != "" {
						local = append(local, result)
					}
				}
			}
			if len(local) > 0 {
				mu.Lock()
				ordered[idx] = local
				mu.Unlock()
			}
			return nil
		})
	}
	_ = gr.Wait()

	results := make([]string, 0, len(paths))
	for _, local := range ordered {
		results = append(results, local...)
	}
	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return "# From:" + filePath + "\n" + string(content)
}
