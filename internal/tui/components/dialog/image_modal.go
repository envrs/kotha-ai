package dialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/tui/image"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/theme"
	"github.com/kothagpt/kotha/internal/tui/util"
)

// ImageEnlargeMsg requests opening an image in the enlarged modal view.
type ImageEnlargeMsg struct {
	Data   []byte
	Format string
}

// CloseImageModalMsg closes the image modal
type CloseImageModalMsg struct{}

// ImageModalCmp interface for the image modal component
type ImageModalCmp interface {
	tea.Model
	layout.Bindings
	SetImage(data []byte, format string) tea.Cmd
}

// ImageModalKeys defines key bindings for the image modal
type ImageModalKeys struct {
	Close key.Binding
	ZoomIn  key.Binding
	ZoomOut key.Binding
	PanUp    key.Binding
	PanDown  key.Binding
	PanLeft  key.Binding
	PanRight key.Binding
	Reset    key.Binding
}

var imageModalKeys = ImageModalKeys{
	Close: key.NewBinding(
		key.WithKeys("esc", "q", "enter"),
		key.WithHelp("esc/q/enter", "close"),
	),
	ZoomIn: key.NewBinding(
		key.WithKeys("+", "="),
		key.WithHelp("+/-", "zoom"),
	),
	ZoomOut: key.NewBinding(
		key.WithKeys("-", "_"),
		key.WithHelp("-/_", "zoom out"),
	),
	PanUp: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "pan up"),
	),
	PanDown: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "pan down"),
	),
	PanLeft: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "pan left"),
	),
	PanRight: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "pan right"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset"),
	),
}

// imageModalCmp is the implementation of ImageModalCmp
type imageModalCmp struct {
	width      int
	height     int
	imageData  []byte
	imageFormat string
	zoom       float64
	panX       int
	panY       int
	showing    bool
}

// NewImageModalCmp creates a new image modal component
func NewImageModalCmp() ImageModalCmp {
	return &imageModalCmp{
		zoom: 1.0,
	}
}

func (m *imageModalCmp) Init() tea.Cmd {
	return nil
}

func (m *imageModalCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "enter":
			return m, util.CmdHandler(CloseImageModalMsg{})
		case "+", "=":
			m.zoom = min(m.zoom*1.25, 8.0)
			return m, nil
		case "-", "_":
			m.zoom = max(m.zoom*0.8, 0.25)
			return m, nil
		case "up":
			m.panY -= 20
			return m, nil
		case "down":
			m.panY += 20
			return m, nil
		case "left":
			m.panX -= 20
			return m, nil
		case "right":
			m.panX += 20
			return m, nil
		case "r":
			m.zoom = 1.0
			m.panX = 0
			m.panY = 0
			return m, nil
		}
	case ImageEnlargeMsg:
		m.imageData = msg.Data
		m.imageFormat = msg.Format
		m.showing = true
		m.zoom = 1.0
		m.panX = 0
		m.panY = 0
		return m, nil
	case CloseImageModalMsg:
		m.showing = false
		m.imageData = nil
		return m, nil
	}
	return m, nil
}

func (m *imageModalCmp) View() string {
	if !m.showing || len(m.imageData) == 0 {
		return ""
	}

	t := theme.CurrentTheme()
	width := m.width - 4
	height := m.height - 4

	// Decode image
	img, _, err := image.DecodeFromBytes(m.imageData)
	if err != nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(t.Error()).
			Render("Failed to decode image: " + err.Error())
	}

	// Generate preview with zoom and pan
	preview := image.ToString(int(float64(width)*m.zoom), img)

	// Apply pan
	lines := splitLines(preview)
	if m.panY > 0 && m.panY < len(lines) {
		lines = lines[m.panY:]
	}
	if m.panX > 0 {
		for i := range lines {
			if len(lines[i]) > m.panX {
				lines[i] = lines[i][m.panX:]
			}
		}
		preview = joinLines(lines)
	}

	// Render modal
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocused()).
		Background(t.Background()).
		Foreground(t.Text()).
		Padding(1)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary()).
		Render("Image Preview (" + m.imageFormat + ")  " + lipgloss.NewStyle().Foreground(t.TextMuted()).Render("esc/q/enter close  +/- zoom  arrows pan  r reset"))

	content := lipgloss.JoinVertical(lipgloss.Left, title, preview)

	return layout.PlaceOverlay(
		m.width/2-lipgloss.Width(style.Render(content))/2,
		m.height/2-lipgloss.Height(style.Render(content))/2,
		style.Render(content),
		strings.Repeat(" ", m.width),
		true,
	)
}

func (m *imageModalCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		imageModalKeys.Close,
		imageModalKeys.ZoomIn,
		imageModalKeys.ZoomOut,
		imageModalKeys.PanUp,
		imageModalKeys.PanDown,
		imageModalKeys.PanLeft,
		imageModalKeys.PanRight,
		imageModalKeys.Reset,
	}
}

func (m *imageModalCmp) SetImage(data []byte, format string) tea.Cmd {
	m.imageData = data
	m.imageFormat = format
	m.showing = true
	m.zoom = 1.0
	m.panX = 0
	m.panY = 0
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	return lines
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}