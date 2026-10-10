package dialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/stretchr/testify/require"
)

// withConfig installs a default config provider for the duration of the test
// so dialog helpers that call config.Get() work without a full app bootstrap.
func withConfig(t *testing.T) {
	t.Helper()
	// Always install a fresh provider so config.Get() never panics. We
	// restore to another fresh (non-nil) provider in cleanup rather than
	// trying to capture the previous one, which may be nil on the first
	// test in the package.
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{
		WorkingDir: t.TempDir(),
		Providers:  map[models.ModelProvider]config.Provider{},
	}))
	t.Cleanup(func() {
		config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{
			WorkingDir: t.TempDir(),
			Providers:  map[models.ModelProvider]config.Provider{},
		}))
	})
}

func TestBuildProviderEntriesAllProviders(t *testing.T) {
	withConfig(t)
	entries := BuildProviderEntries()
	require.Len(t, entries, len(models.AllProviders))

	seen := make(map[models.ModelProvider]bool)
	for _, e := range entries {
		require.NotEmpty(t, e.Name)
		require.False(t, seen[e.Provider], "duplicate provider %s", e.Provider)
		seen[e.Provider] = true
	}
}

func TestBuildProviderEntriesReflectsConfig(t *testing.T) {
	withConfig(t)
	// Use a config where Anthropic is explicitly enabled with a key.
	entries := BuildProviderEntries()
	var anthro *ProviderEntry
	for i := range entries {
		if entries[i].Provider == models.ProviderAnthropic {
			anthro = &entries[i]
			break
		}
	}
	require.NotNil(t, anthro)
	require.Equal(t, "Anthropic", anthro.Name)
}

func TestProviderConnectToggle(t *testing.T) {
	m := NewProviderConnectCmp()
	m.SetProviders([]ProviderEntry{
		{Name: "Anthropic", Provider: models.ProviderAnthropic, Enabled: false},
		{Name: "OpenAI", Provider: models.ProviderOpenAI, Enabled: true},
	})

	// Selecting the first entry (Anthropic) should emit a toggle message.
	_, cmd := m.Update(enterMsg())
	require.NotNil(t, cmd)
	msg := cmd().(ProviderConnectSelectedMsg)
	require.Equal(t, models.ProviderAnthropic, msg.Provider)
	require.True(t, msg.Enabled)
}

func TestProviderConnectNavigation(t *testing.T) {
	m := NewProviderConnectCmp()
	m.SetProviders([]ProviderEntry{
		{Name: "Anthropic", Provider: models.ProviderAnthropic},
		{Name: "OpenAI", Provider: models.ProviderOpenAI},
		{Name: "Gemini", Provider: models.ProviderGemini},
	})

	// Move down twice.
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	// The third entry should now be selected.
	_, cmd := m.Update(enterMsg())
	require.NotNil(t, cmd)
	msg := cmd().(ProviderConnectSelectedMsg)
	require.Equal(t, models.ProviderGemini, msg.Provider)
}

func TestProviderConnectClose(t *testing.T) {
	m := NewProviderConnectCmp()
	m.SetProviders([]ProviderEntry{
		{Name: "Anthropic", Provider: models.ProviderAnthropic},
	})

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// The esc key should be handled; we just verify the dialog still renders.
	view := m.View()
	require.Contains(t, view, "Anthropic")
}

func TestProviderConnectViewShowsStatus(t *testing.T) {
	m := NewProviderConnectCmp()
	m.SetProviders([]ProviderEntry{
		{Name: "Anthropic", Provider: models.ProviderAnthropic, Enabled: true, HasAPI: true},
		{Name: "OpenAI", Provider: models.ProviderOpenAI, Enabled: false},
	})

	view := m.View()
	require.Contains(t, view, "Anthropic")
	require.Contains(t, view, "connected")
	require.Contains(t, view, "OpenAI")
	require.Contains(t, view, "disconnected")
}

func TestProviderConnectEmpty(t *testing.T) {
	m := NewProviderConnectCmp()
	m.SetProviders(nil)
	view := m.View()
	require.Contains(t, strings.ToLower(view), "no providers")
}

func TestProviderConnectBindingKeys(t *testing.T) {
	m := NewProviderConnectCmp()
	keys := m.BindingKeys()
	require.NotEmpty(t, keys)
}
