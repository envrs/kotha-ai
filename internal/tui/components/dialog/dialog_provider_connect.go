package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
)

// ProviderEntry describes a provider that can be connected/disconnected.
type ProviderEntry struct {
	Name     string
	Provider models.ModelProvider
	Enabled  bool
	HasAPI   bool
	HasEnv   bool
	Editing  bool
	APIKey   string
}

// ProviderConnectSelectedMsg is emitted when the user toggles a provider's
// connect state.
type ProviderConnectSelectedMsg struct {
	Provider models.ModelProvider
	Enabled  bool
	APIKey   string
}

// CloseProviderConnectDialogMsg closes the dialog.
type CloseProviderConnectDialogMsg struct{}

// ShowProviderConnectDialogMsg requests the dialog be shown.
type ShowProviderConnectDialogMsg struct{}

// ProviderConnectDialog is the interface for the provider connect dialog.
type ProviderConnectDialog interface {
	tea.Model
	layout.Bindings
	SetProviders(providers []ProviderEntry)
}

type providerConnectCmp struct {
	providers []ProviderEntry
	idx       int
	width     int
}

var providerConnectKeys = struct {
	Up, Down, Toggle, Close key.Binding
}{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "previous provider"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next provider"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "toggle connect"),
	),
	Close: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close"),
	),
}

func (c *providerConnectCmp) Init() tea.Cmd { return nil }

func (c *providerConnectCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, providerConnectKeys.Up):
			if c.idx > 0 {
				c.idx--
			}
		case key.Matches(msg, providerConnectKeys.Down):
			if c.idx < len(c.providers)-1 {
				c.idx++
			}
		case key.Matches(msg, providerConnectKeys.Toggle):
			if len(c.providers) > 0 {
				p := c.providers[c.idx]
				return c, func() tea.Msg {
					return ProviderConnectSelectedMsg{
						Provider: p.Provider,
						Enabled:  !p.Enabled,
						APIKey:   p.APIKey,
					}
				}
			}
		case key.Matches(msg, providerConnectKeys.Close):
			return c, func() tea.Msg { return CloseProviderConnectDialogMsg{} }
		}
	}
	return c, nil
}

func (c *providerConnectCmp) View() string {
	t := theme.CurrentTheme()
	base := styles.BaseStyle()

	out := base.Bold(true).Foreground(t.Primary()).Render("Connect provider") + "\n"
	out += base.Foreground(t.TextMuted()).Render("Toggle a provider to enable or disable it.") + "\n\n"

	if len(c.providers) == 0 {
		out += base.Foreground(t.TextMuted()).Render("  no providers available") + "\n"
		return out
	}

	maxLabel := 0
	for _, p := range c.providers {
		if w := lipgloss.Width(p.Name); w > maxLabel {
			maxLabel = w
		}
	}

	for i, p := range c.providers {
		label := p.Name
		if lipgloss.Width(label) < maxLabel {
			label += strings.Repeat(" ", maxLabel-lipgloss.Width(label))
		}

		var mark string
		var markStyle lipgloss.Style
		if p.Enabled {
			mark = "● connected"
			markStyle = base.Foreground(t.Success()).Bold(true)
		} else {
			mark = "○ disconnected"
			markStyle = base.Foreground(t.TextMuted())
		}

		line := fmt.Sprintf("%s  %s", label, markStyle.Render(mark))

		// Show credential source hint.
		var hint string
		switch {
		case p.HasAPI:
			hint = base.Foreground(t.TextMuted()).Render("(configured)")
		case p.HasEnv:
			hint = base.Foreground(t.Info()).Render("(env)")
		default:
			hint = base.Foreground(t.Warning()).Render("(no key)")
		}

		if i == c.idx {
			out += base.Background(t.Primary()).Foreground(t.Background()).Render("▸ "+line+" "+hint) + "\n"
		} else {
			out += "  " + line + "  " + hint + "\n"
		}
	}

	out += "\n" + base.Foreground(t.TextMuted()).Render("esc close • enter toggle")
	return out
}

func (c *providerConnectCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(providerConnectKeys)
}

func (c *providerConnectCmp) SetProviders(providers []ProviderEntry) {
	c.providers = providers
	c.idx = 0
}

// NewProviderConnectCmp creates a new provider connect dialog.
func NewProviderConnectCmp() ProviderConnectDialog {
	return &providerConnectCmp{}
}

// BuildProviderEntries builds the list of provider entries from the current
// configuration. It is used by the TUI to populate the dialog.
func BuildProviderEntries() []ProviderEntry {
	cfg := config.Get()
	if cfg == nil {
		return nil
	}

	var entries []ProviderEntry
	for _, p := range models.AllProviders {
		providerCfg, exists := cfg.Providers[p]
		enabled := exists && !providerCfg.Disabled && providerCfg.APIKey != ""
		entries = append(entries, ProviderEntry{
			Name:     models.ProviderDisplayName[p],
			Provider: p,
			Enabled:  enabled,
			HasAPI:   exists && providerCfg.APIKey != "",
			HasEnv:   models.ProviderHasCredentials(p),
			APIKey:   providerCfg.APIKey,
		})
	}
	return entries
}
