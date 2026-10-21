package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kothagpt/kotha/internal/lsp"
	"github.com/kothagpt/kotha/internal/lsp/protocol"
	"github.com/kothagpt/kotha/internal/symbol"
)

// IndexSymbols walks the workspace and extracts symbols from every source file
// using the given LSP client, persisting them through the symbol service.
func (app *App) IndexSymbols(ctx context.Context, name string, client *lsp.Client) error {
	if client == nil {
		return fmt.Errorf("nil LSP client")
	}
	workDir := app.configProvider.WorkingDirectory()
	symbols, err := app.ExtractSymbolsFromDir(ctx, client, workDir)
	if err != nil {
		return err
	}
	for _, sym := range symbols {
		if _, err := app.Symbols.Create(ctx, sym); err != nil {
			return fmt.Errorf("persist symbol %s: %w", sym.Name, err)
		}
	}
	return nil
}

func shouldSkipDir(path string) bool {
	dirName := filepath.Base(path)
	if strings.HasPrefix(dirName, ".") {
		return true
	}
	skipDirs := map[string]bool{
		"node_modules": true,
		"dist":         true,
		"build":        true,
		"coverage":     true,
		"vendor":       true,
		"target":       true,
	}
	return skipDirs[dirName]
}

func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rs", ".c", ".cpp", ".h", ".hpp", ".java", ".rb", ".php", ".swift", ".kt":
		return true
	}
	return false
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".jsx":
		return "jsx"
	case ".rs":
		return "rust"
	case ".c", ".cpp", ".h", ".hpp":
		return "c"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	}
	return ext
}

// ExtractSymbolsFromDir walks a directory tree and returns symbols for each
// source file that the LSP client can handle.
func (app *App) ExtractSymbolsFromDir(ctx context.Context, client *lsp.Client, root string) ([]symbol.Symbol, error) {
	var out []symbol.Symbol
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFile(path) {
			return nil
		}
		syms, err := app.ExtractSymbolsFromFile(ctx, client, path)
		if err != nil {
			return nil // tolerate per-file failures
		}
		out = append(out, syms...)
		return nil
	})
	return out, err
}

// ExtractSymbolsFromFile opens a file with the LSP client and returns its
// document symbols, mapped to the symbol service's domain type.
func (app *App) ExtractSymbolsFromFile(ctx context.Context, client *lsp.Client, filePath string) ([]symbol.Symbol, error) {
	if err := client.OpenFile(ctx, filePath); err != nil {
		return nil, err
	}
	defer func() { _ = client.CloseFile(ctx, filePath) }()

	var result protocol.Or_Result_textDocument_documentSymbol
	result, err := client.DocumentSymbol(ctx, protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentUri("file://" + filePath),
		},
	})
	if err != nil {
		return nil, err
	}
	_ = result

	lang := detectLanguage(filePath)
	var out []symbol.Symbol
	results, err := result.Results()
	if err != nil {
		return nil, err
	}
	for _, sym := range results {
		name := sym.GetName()
		rng := sym.GetRange()
		container := ""
		kind := ""
		if si, ok := sym.(*protocol.SymbolInformation); ok {
			kind = strconv.FormatUint(uint64(si.Kind), 10)
			container = si.ContainerName
		}
		out = append(out, symbol.Symbol{
			FilePath:      filePath,
			Name:          name,
			Kind:          kind,
			Range:         fmt.Sprintf("%d:%d-%d:%d", rng.Start.Line, rng.Start.Character, rng.End.Line, rng.End.Character),
			ContainerName: container,
			Language:      lang,
		})
	}
	return out, nil
}