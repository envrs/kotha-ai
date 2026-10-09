package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYamlThemeParsing(t *testing.T) {
	data := []byte(`
accent: "#7cafc2"
background: "#181818"
details: darker
foreground: "#d8d8d8"
terminal_colors:
  bright:
    black: "#585858"
    blue: "#7cafc2"
    cyan: "#86c1b9"
    green: "#a1b56c"
    magenta: "#ba8baf"
    red: "#ab4642"
    white: "#f8f8f8"
    yellow: "#f7ca88"
  normal:
    black: "#181818"
    blue: "#7cafc2"
    cyan: "#86c1b9"
    green: "#a1b56c"
    magenta: "#ba8baf"
    red: "#ab4642"
    white: "#d8d8d8"
    yellow: "#f7ca88"
`)
	def, err := ParseYamlTheme(data)
	if err != nil {
		t.Fatalf("Failed to parse YAML theme: %v", err)
	}
	if def.Accent != "#7cafc2" {
		t.Errorf("Expected accent #7cafc2, got %s", def.Accent)
	}
	if def.Background != "#181818" {
		t.Errorf("Expected background #181818, got %s", def.Background)
	}
	if def.TerminalColors.Normal.Red != "#ab4642" {
		t.Errorf("Expected normal red #ab4642, got %s", def.TerminalColors.Normal.Red)
	}
	if def.TerminalColors.Bright.Black != "#585858" {
		t.Errorf("Expected bright black #585858, got %s", def.TerminalColors.Bright.Black)
	}
}

func TestNewYamlThemeFromDef(t *testing.T) {
	def := &YamlThemeDef{
		Accent:     "#7cafc2",
		Background: "#181818",
		Details:    "darker",
		Foreground: "#d8d8d8",
		TerminalColors: ColorPalette{
			Normal: TerminalColors{
				Black:   "#181818",
				Blue:    "#7cafc2",
				Cyan:    "#86c1b9",
				Green:   "#a1b56c",
				Magenta: "#ba8baf",
				Red:     "#ab4642",
				White:   "#d8d8d8",
				Yellow:  "#f7ca88",
			},
			Bright: TerminalColors{
				Black:   "#585858",
				Blue:    "#7cafc2",
				Cyan:    "#86c1b9",
				Green:   "#a1b56c",
				Magenta: "#ba8baf",
				Red:     "#ab4642",
				White:   "#f8f8f8",
				Yellow:  "#f7ca88",
			},
		},
	}

	yt := NewYamlTheme(def)
	if yt == nil {
		t.Fatal("Expected YamlTheme, got nil")
	}

	// Spot check derived colors
	if yt.Background().Dark != "#181818" {
		t.Errorf("Expected background #181818, got %s", yt.Background().Dark)
	}
	if yt.Text().Dark != "#d8d8d8" {
		t.Errorf("Expected text #d8d8d8, got %s", yt.Text().Dark)
	}
	if yt.Error().Dark != "#ab4642" {
		t.Errorf("Expected error #ab4642, got %s", yt.Error().Dark)
	}
	if yt.Success().Dark != "#a1b56c" {
		t.Errorf("Expected success #a1b56c, got %s", yt.Success().Dark)
	}
}

func TestLoadYamlThemesFromDir(t *testing.T) {
	tmp := t.TempDir()

	yamlContent := `
accent: "#ff5500"
background: "#1a1a2e"
details: darker
foreground: "#e0e0e0"
terminal_colors:
  normal:
    black: "#1a1a2e"
    blue: "#00aaff"
    cyan: "#00ffcc"
    green: "#00cc00"
    magenta: "#cc00cc"
    red: "#cc0000"
    white: "#e0e0e0"
    yellow: "#cccc00"
  bright:
    black: "#444444"
    blue: "#00aaff"
    cyan: "#00ffcc"
    green: "#00cc00"
    magenta: "#cc00cc"
    red: "#cc0000"
    white: "#ffffff"
    yellow: "#cccc00"
`
	if err := os.WriteFile(filepath.Join(tmp, "test_theme.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("Failed to write test theme: %v", err)
	}

	loaded, err := LoadYamlThemesFromDir(tmp)
	if err != nil {
		t.Fatalf("LoadYamlThemesFromDir failed: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != "test_theme" {
		t.Errorf("Expected [test_theme], got %v", loaded)
	}

	// Verify the theme was registered
	tm := GetTheme("test_theme")
	if tm == nil {
		t.Fatal("Expected test_theme to be registered")
	}
	if tm.Background().Dark != "#1a1a2e" {
		t.Errorf("Expected background #1a1a2e, got %s", tm.Background().Dark)
	}
}
