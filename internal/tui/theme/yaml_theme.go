package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// TerminalColors holds the 8 standard ANSI color groups.
type TerminalColors struct {
	Black   string `yaml:"black"`
	Blue    string `yaml:"blue"`
	Cyan    string `yaml:"cyan"`
	Green   string `yaml:"green"`
	Magenta string `yaml:"magenta"`
	Red     string `yaml:"red"`
	White   string `yaml:"white"`
	Yellow  string `yaml:"yellow"`
}

// ColorPalette has normal and bright variants of the 8 ANSI colors.
type ColorPalette struct {
	Normal TerminalColors `yaml:"normal"`
	Bright TerminalColors `yaml:"bright"`
}

// YamlThemeDef is the parsed YAML representation of a theme file.
type YamlThemeDef struct {
	Accent         string       `yaml:"accent"`
	Background     string       `yaml:"background"`
	Details        string       `yaml:"details"`
	Foreground     string       `yaml:"foreground"`
	TerminalColors ColorPalette `yaml:"terminal_colors"`

	// Optional overrides for all Theme interface colors.
	// Any that are set will take precedence over the derived values.
	Primary             *string `yaml:"primary,omitempty"`
	Secondary           *string `yaml:"secondary,omitempty"`
	Error               *string `yaml:"error,omitempty"`
	Warning             *string `yaml:"warning,omitempty"`
	Success             *string `yaml:"success,omitempty"`
	Info                *string `yaml:"info,omitempty"`
	Text                *string `yaml:"text,omitempty"`
	TextMuted           *string `yaml:"textMuted,omitempty"`
	TextEmphasized      *string `yaml:"textEmphasized,omitempty"`
	BackgroundSecondary *string `yaml:"backgroundSecondary,omitempty"`
	BackgroundDarker    *string `yaml:"backgroundDarker,omitempty"`
	BorderNormal        *string `yaml:"borderNormal,omitempty"`
	BorderFocused       *string `yaml:"borderFocused,omitempty"`
	BorderDim           *string `yaml:"borderDim,omitempty"`

	DiffAdded               *string `yaml:"diffAdded,omitempty"`
	DiffRemoved             *string `yaml:"diffRemoved,omitempty"`
	DiffContext             *string `yaml:"diffContext,omitempty"`
	DiffHunkHeader          *string `yaml:"diffHunkHeader,omitempty"`
	DiffHighlightAdded      *string `yaml:"diffHighlightAdded,omitempty"`
	DiffHighlightRemoved    *string `yaml:"diffHighlightRemoved,omitempty"`
	DiffAddedBg             *string `yaml:"diffAddedBg,omitempty"`
	DiffRemovedBg           *string `yaml:"diffRemovedBg,omitempty"`
	DiffContextBg           *string `yaml:"diffContextBg,omitempty"`
	DiffLineNumber          *string `yaml:"diffLineNumber,omitempty"`
	DiffAddedLineNumberBg   *string `yaml:"diffAddedLineNumberBg,omitempty"`
	DiffRemovedLineNumberBg *string `yaml:"diffRemovedLineNumberBg,omitempty"`

	MarkdownText            *string `yaml:"markdownText,omitempty"`
	MarkdownHeading         *string `yaml:"markdownHeading,omitempty"`
	MarkdownLink            *string `yaml:"markdownLink,omitempty"`
	MarkdownLinkText        *string `yaml:"markdownLinkText,omitempty"`
	MarkdownCode            *string `yaml:"markdownCode,omitempty"`
	MarkdownBlockQuote      *string `yaml:"markdownBlockQuote,omitempty"`
	MarkdownEmph            *string `yaml:"markdownEmph,omitempty"`
	MarkdownStrong          *string `yaml:"markdownStrong,omitempty"`
	MarkdownHorizontalRule  *string `yaml:"markdownHorizontalRule,omitempty"`
	MarkdownListItem        *string `yaml:"markdownListItem,omitempty"`
	MarkdownListEnumeration *string `yaml:"markdownListEnumeration,omitempty"`
	MarkdownImage           *string `yaml:"markdownImage,omitempty"`
	MarkdownImageText       *string `yaml:"markdownImageText,omitempty"`
	MarkdownCodeBlock       *string `yaml:"markdownCodeBlock,omitempty"`

	SyntaxComment     *string `yaml:"syntaxComment,omitempty"`
	SyntaxKeyword     *string `yaml:"syntaxKeyword,omitempty"`
	SyntaxFunction    *string `yaml:"syntaxFunction,omitempty"`
	SyntaxVariable    *string `yaml:"syntaxVariable,omitempty"`
	SyntaxString      *string `yaml:"syntaxString,omitempty"`
	SyntaxNumber      *string `yaml:"syntaxNumber,omitempty"`
	SyntaxType        *string `yaml:"syntaxType,omitempty"`
	SyntaxOperator    *string `yaml:"syntaxOperator,omitempty"`
	SyntaxPunctuation *string `yaml:"syntaxPunctuation,omitempty"`
}

// YamlTheme implements the Theme interface from a parsed YAML definition.
type YamlTheme struct {
	BaseTheme
}

// yamlColor picks the override if set, otherwise falls back to the derived value.
func pick(override *string, derived string) string {
	if override != nil && *override != "" {
		return *override
	}
	return derived
}

// deriveColor is a helper that darkens or lightens a hex color by a factor.
// factor > 0 lightens (mix toward white), factor < 0 darkens (mix toward black).
func deriveColor(hex string, factor float64) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return hex
	}
	r, g, b :=
		hexByte(hex[0:2]),
		hexByte(hex[2:4]),
		hexByte(hex[4:6])
	if factor >= 0 {
		r = int(float64(r) + (255-float64(r))*factor)
		g = int(float64(g) + (255-float64(g))*factor)
		b = int(float64(b) + (255-float64(b))*factor)
	} else {
		f := 1 + factor // factor is negative
		r = int(float64(r) * f)
		g = int(float64(g) * f)
		b = int(float64(b) * f)
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hexByte(s string) int {
	v, _ := strconv.ParseInt(s, 16, 32)
	return int(v)
}

// NewYamlTheme creates a Theme from a parsed YAML definition.
// It maps the simplified YAML fields (accent, background, foreground,
// terminal_colors) to the full Theme interface using sensible mappings,
// with optional per-field overrides from the YAML.
func NewYamlTheme(def *YamlThemeDef) *YamlTheme {
	t := &YamlTheme{}
	n := def.TerminalColors.Normal
	b := def.TerminalColors.Bright

	isDarker := strings.EqualFold(def.Details, "darker")

	// Derived background shades
	bg := def.Background
	var bgSecondary, bgDarker string
	if isDarker {
		bgSecondary = deriveColor(bg, 0.05)
		bgDarker = deriveColor(bg, -0.3)
	} else {
		bgSecondary = deriveColor(bg, -0.05)
		bgDarker = deriveColor(bg, 0.2)
	}

	// Use the accent for primary, or derive from blue if absent
	accent := def.Accent
	if accent == "" {
		accent = n.Blue
	}

	text := def.Foreground
	if text == "" {
		text = n.White
	}

	// Terminal color → Theme mapping
	errColor := pick(def.Error, n.Red)
	warnColor := pick(def.Warning, n.Yellow)
	okColor := pick(def.Success, n.Green)
	infoColor := pick(def.Info, n.Blue)
	primary := pick(def.Primary, accent)
	secondary := pick(def.Secondary, n.Magenta)
	muted := pick(def.TextMuted, n.Black)
	if muted == "" || muted == bg {
		// black often equals background; use a mid-gray instead
		muted = deriveColor(text, -0.5)
	}
	emph := pick(def.TextEmphasized, b.White)

	// Base colors
	t.PrimaryColor = adaptive(primary)
	t.SecondaryColor = adaptive(secondary)
	t.AccentColor = adaptive(accent)

	// Status colors
	t.ErrorColor = adaptive(errColor)
	t.WarningColor = adaptive(warnColor)
	t.SuccessColor = adaptive(okColor)
	t.InfoColor = adaptive(infoColor)

	// Text colors
	t.TextColor = adaptive(text)
	t.TextMutedColor = adaptive(muted)
	t.TextEmphasizedColor = adaptive(emph)

	// Background colors
	t.BackgroundColor = adaptive(bg)
	t.BackgroundSecondaryColor = adaptive(pick(def.BackgroundSecondary, bgSecondary))
	t.BackgroundDarkerColor = adaptive(pick(def.BackgroundDarker, bgDarker))

	// Border colors
	t.BorderNormalColor = adaptive(pick(def.BorderNormal, n.Black))
	t.BorderFocusedColor = adaptive(pick(def.BorderFocused, primary))
	t.BorderDimColor = adaptive(pick(def.BorderDim, n.Black))

	// Diff view colors
	t.DiffAddedColor = adaptive(pick(def.DiffAdded, okColor))
	t.DiffRemovedColor = adaptive(pick(def.DiffRemoved, errColor))
	t.DiffContextColor = adaptive(pick(def.DiffContext, muted))
	t.DiffHunkHeaderColor = adaptive(pick(def.DiffHunkHeader, secondary))
	t.DiffHighlightAddedColor = adaptive(pick(def.DiffHighlightAdded, n.Green))
	t.DiffHighlightRemovedColor = adaptive(pick(def.DiffHighlightRemoved, n.Red))
	addedBgFactor := -0.5
	if isDarker {
		addedBgFactor = -0.7
	}
	t.DiffAddedBgColor = adaptive(pick(def.DiffAddedBg, deriveColor(okColor, addedBgFactor)))
	t.DiffRemovedBgColor = adaptive(pick(def.DiffRemovedBg, deriveColor(errColor, addedBgFactor)))
	t.DiffContextBgColor = adaptive(pick(def.DiffContextBg, bg))
	t.DiffLineNumberColor = adaptive(pick(def.DiffLineNumber, muted))
	lnumBgFactor := -0.7
	if isDarker {
		lnumBgFactor = -0.8
	}
	t.DiffAddedLineNumberBgColor = adaptive(pick(def.DiffAddedLineNumberBg, deriveColor(okColor, lnumBgFactor)))
	t.DiffRemovedLineNumberBgColor = adaptive(pick(def.DiffRemovedLineNumberBg, deriveColor(errColor, lnumBgFactor)))

	// Markdown colors
	t.MarkdownTextColor = adaptive(pick(def.MarkdownText, text))
	t.MarkdownHeadingColor = adaptive(pick(def.MarkdownHeading, secondary))
	t.MarkdownLinkColor = adaptive(pick(def.MarkdownLink, primary))
	t.MarkdownLinkTextColor = adaptive(pick(def.MarkdownLinkText, n.Cyan))
	t.MarkdownCodeColor = adaptive(pick(def.MarkdownCode, okColor))
	t.MarkdownBlockQuoteColor = adaptive(pick(def.MarkdownBlockQuote, warnColor))
	t.MarkdownEmphColor = adaptive(pick(def.MarkdownEmph, warnColor))
	t.MarkdownStrongColor = adaptive(pick(def.MarkdownStrong, accent))
	t.MarkdownHorizontalRuleColor = adaptive(pick(def.MarkdownHorizontalRule, muted))
	t.MarkdownListItemColor = adaptive(pick(def.MarkdownListItem, primary))
	t.MarkdownListEnumerationColor = adaptive(pick(def.MarkdownListEnumeration, n.Cyan))
	t.MarkdownImageColor = adaptive(pick(def.MarkdownImage, primary))
	t.MarkdownImageTextColor = adaptive(pick(def.MarkdownImageText, n.Cyan))
	t.MarkdownCodeBlockColor = adaptive(pick(def.MarkdownCodeBlock, text))

	// Syntax highlighting colors
	t.SyntaxCommentColor = adaptive(pick(def.SyntaxComment, muted))
	t.SyntaxKeywordColor = adaptive(pick(def.SyntaxKeyword, secondary))
	t.SyntaxFunctionColor = adaptive(pick(def.SyntaxFunction, primary))
	t.SyntaxVariableColor = adaptive(pick(def.SyntaxVariable, errColor))
	t.SyntaxStringColor = adaptive(pick(def.SyntaxString, okColor))
	t.SyntaxNumberColor = adaptive(pick(def.SyntaxNumber, accent))
	t.SyntaxTypeColor = adaptive(pick(def.SyntaxType, warnColor))
	t.SyntaxOperatorColor = adaptive(pick(def.SyntaxOperator, n.Cyan))
	t.SyntaxPunctuationColor = adaptive(pick(def.SyntaxPunctuation, text))

	return t
}

func adaptive(hex string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Dark: hex, Light: hex}
}

// LoadYamlThemeFile reads and parses a YAML theme file.
func LoadYamlThemeFile(path string) (*YamlThemeDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read theme file %s: %w", path, err)
	}
	return ParseYamlTheme(data)
}

// ParseYamlTheme parses a YAML theme definition from raw bytes.
func ParseYamlTheme(data []byte) (*YamlThemeDef, error) {
	var def YamlThemeDef
	// yaml.Unmarshal is defined in the yaml package; imported in the caller file.
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse theme YAML: %w", err)
	}
	return &def, nil
}

// LoadYamlThemesFromDir scans a directory for *.yaml / *.yml files,
// registers each as a theme under its filename (without extension),
// and returns the names of successfully loaded themes.
func LoadYamlThemesFromDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var loaded []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		def, err := LoadYamlThemeFile(path)
		if err != nil {
			continue // skip invalid files
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		RegisterTheme(name, NewYamlTheme(def))
		loaded = append(loaded, name)
	}
	return loaded, nil
}
