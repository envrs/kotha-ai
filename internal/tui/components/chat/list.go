package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/app"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/pubsub"
	"github.com/kothagpt/kotha/internal/session"
	"github.com/kothagpt/kotha/internal/tui/components/dialog"
	"github.com/kothagpt/kotha/internal/tui/styles"
	"github.com/kothagpt/kotha/internal/tui/theme"
	"github.com/kothagpt/kotha/internal/tui/util"
)

type cacheItem struct {
	width   int
	content []uiMessage
}
type messagesCmp struct {
	app           *app.App
	width, height int
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentMsgID  string
	cachedContent map[string]cacheItem
	spinner       spinner.Model
	rendering     bool
	attachments   viewport.Model
	lastRender    time.Time
	pendingRender bool
	toolIndex     map[string]message.ToolResult
	toolGen       int // len(m.messages) when index built; rebuild if changed
	helpCache     string
	helpCacheKey  string
	workCache     string
	workCacheKey  string
	fullContent   string
}
type renderFinishedMsg struct{}
type deferredRenderMsg struct{}

type MessageKeys struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
}

var messageKeys = MessageKeys{
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("f/pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("b/pgup", "page up"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init(), m.spinner.Tick)
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case dialog.ThemeChangedMsg:
		styles.InvalidateMarkdownCache()
		m.rerender()
		return m, nil
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			return m, cmd
		}
		return m, nil
	case SessionClearedMsg:
		m.session = session.Session{}
		m.messages = make([]message.Message, 0)
		m.currentMsgID = ""
		m.rendering = false
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, messageKeys.PageUp) || key.Matches(msg, messageKeys.PageDown) ||
			key.Matches(msg, messageKeys.HalfPageUp) || key.Matches(msg, messageKeys.HalfPageDown) {
			u, cmd := m.viewport.Update(msg)
			m.viewport = u
			cmds = append(cmds, cmd)
		}

	case renderFinishedMsg:
		m.rendering = false
		m.viewport.GotoBottom()
	case deferredRenderMsg:
		m.pendingRender = false
		m.renderView()
		if len(m.messages) > 0 {
			m.viewport.GotoBottom()
		}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.session.ID {
			m.session = msg.Payload
			if m.session.SummaryMessageID == m.currentMsgID {
				delete(m.cachedContent, m.currentMsgID)
				m.renderView()
			}
		}
	case pubsub.Event[message.Message]:
		needsRerender := false
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {

				messageExists := false
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						messageExists = true
						break
					}
				}

				if !messageExists {
					if len(m.messages) > 0 {
						lastMsgID := m.messages[len(m.messages)-1].ID
						delete(m.cachedContent, lastMsgID)
					}

					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					needsRerender = true
				}
			}
			// There are tool calls from the child task
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					if c.ID == msg.Payload.SessionID {
						delete(m.cachedContent, v.ID)
						needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					needsRerender = true
					break
				}
			}
		}
		if needsRerender {
			// Throttle streaming token updates: coalesce bursts into
			// at most ~20fps full re-renders.
			if time.Since(m.lastRender) < 50*time.Millisecond {
				if !m.pendingRender {
					m.pendingRender = true
					cmds = append(cmds, tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
						return deferredRenderMsg{}
					}))
				}
			} else {
				m.lastRender = time.Now()
				m.renderView()
				if len(m.messages) > 0 {
					if (msg.Type == pubsub.CreatedEvent) ||
						(msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.messages[len(m.messages)-1].ID) {
						m.viewport.GotoBottom()
					}
				}
			}
		}
	}

	// Only animate spinner while work is in flight; otherwise every
	// spinner.Tick (~60fps) forces a full View() for nothing.
	if m.IsAgentWorking() {
		spinner, cmd := m.spinner.Update(msg)
		m.spinner = spinner
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *messagesCmp) IsAgentWorking() bool {
	return m.app.CoderAgent.IsSessionBusy(m.session.ID)
}

func (m *messagesCmp) renderView() {
	// Rebuild tool index once per render instead of O(N*M) per tool call.
	if m.toolIndex == nil || m.toolGen != len(m.messages) {
		m.toolIndex = BuildToolIndex(m.messages)
		m.toolGen = len(m.messages)
	}
	// Invalidate cached status lines; message set changed.
	m.workCacheKey = ""
	m.helpCacheKey = ""
	m.uiMessages = make([]uiMessage, 0, len(m.messages)*2)
	pos := 0
	baseStyle := styles.BaseStyle()

	if m.width == 0 {
		return
	}
	for inx, msg := range m.messages {
		switch msg.Role {
		case message.User:
			if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width {
				m.uiMessages = append(m.uiMessages, cache.content...)
				for _, um := range cache.content {
					pos += um.height + 1 // + 1 for spacing
				}
				continue
			}
			userMsg := renderUserMessage(
				msg,
				msg.ID == m.currentMsgID,
				m.width,
				pos,
			)
			m.uiMessages = append(m.uiMessages, userMsg)
			m.cachedContent[msg.ID] = cacheItem{
				width:   m.width,
				content: []uiMessage{userMsg},
			}
			pos += userMsg.height + 1 // + 1 for spacing
		case message.Assistant:
			if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width {
				m.uiMessages = append(m.uiMessages, cache.content...)
				for _, um := range cache.content {
					pos += um.height + 1
				}
				continue
			}
			isSummary := m.session.SummaryMessageID == msg.ID

			assistantMessages := renderAssistantMessage(
				msg,
				inx,
				m.messages,
				m.toolIndex,
				m.app.Messages,
				m.currentMsgID,
				isSummary,
				m.width,
				pos,
			)
			for _, msg := range assistantMessages {
				m.uiMessages = append(m.uiMessages, msg)
				pos += msg.height + 1 // + 1 for spacing
			}
			m.cachedContent[msg.ID] = cacheItem{
				width:   m.width,
				content: assistantMessages,
			}
		}
	}

	messages := make([]string, 0, len(m.uiMessages)*2)
	for _, v := range m.uiMessages {
		messages = append(messages, lipgloss.JoinVertical(lipgloss.Left, v.content),
			baseStyle.
				Width(m.width).
				Render(
					"",
				),
		)
	}

	content := baseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				messages...,
			),
		)
	// Skip viewport re-layout when nothing changed (e.g. throttled tick).
	if content == m.fullContent {
		return
	}
	m.fullContent = content
	m.viewport.SetContent(content)
}

func (m *messagesCmp) View() string {
	baseStyle := styles.BaseStyle()

	if m.rendering {
		return baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					"Loading...",
					m.working(),
					m.help(),
				),
			)
	}
	if len(m.messages) == 0 {
		content := baseStyle.
			Width(m.width).
			Height(m.height - 1).
			Render(
				m.initialScreen(),
			)

		return baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					content,
					"",
					m.help(),
				),
			)
	}

	return baseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.viewport.View(),
				m.working(),
				m.help(),
			),
		)
}

func (m *messagesCmp) working() string {
	busy := m.IsAgentWorking()
	key := fmt.Sprintf("%v|%d|%s|%d", busy, len(m.messages), m.spinner.View(), m.width)
	if key == m.workCacheKey {
		return m.workCache
	}
	text := ""
	if busy && len(m.messages) > 0 {
		t := theme.CurrentTheme()
		baseStyle := styles.BaseStyle()

		task := "Thinking..."
		lastMessage := m.messages[len(m.messages)-1]
		waiting, building := ToolState(m.messages)
		if waiting {
			task = "Waiting for tool response..."
		} else if building {
			task = "Building tool call..."
		} else if !lastMessage.IsFinished() {
			task = "Generating..."
		}
		if task != "" {
			text += baseStyle.
				Width(m.width).
				Foreground(t.Primary()).
				Bold(true).
				Render(fmt.Sprintf("%s %s ", m.spinner.View(), task))
		}
	}
	m.workCache = text
	m.workCacheKey = key
	return text
}

func (m *messagesCmp) help() string {
	busy := m.app.CoderAgent.IsBusy()
	key := fmt.Sprintf("%v|%d", busy, m.width)
	if key == m.helpCacheKey {
		return m.helpCache
	}
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	text := ""

	if busy {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("esc"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to exit cancel"),
		)
	} else {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("enter"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to send the message,"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" write"),
			baseStyle.Foreground(t.Text()).Bold(true).Render(" \\"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" and enter to add a new line"),
		)
	}
	out := baseStyle.
		Width(m.width).
		Render(text)
	m.helpCache = out
	m.helpCacheKey = key
	return out
}

func (m *messagesCmp) initialScreen() string {
	baseStyle := styles.BaseStyle()

	return baseStyle.Width(m.width).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			header(m.width),
			"",
			lspsConfigured(m.width),
		),
	)
}

func (m *messagesCmp) rerender() {
	for _, msg := range m.messages {
		delete(m.cachedContent, msg.ID)
	}
	m.renderView()
}

func (m *messagesCmp) SetSize(width, height int) tea.Cmd {
	if m.width == width && m.height == height {
		return nil
	}
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 2
	m.attachments.Width = width + 40
	m.attachments.Height = 3
	m.rerender()
	return nil
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) SetSession(session session.Session) tea.Cmd {
	if m.session.ID == session.ID {
		return nil
	}
	m.session = session
	messages, err := m.app.Messages.List(context.Background(), session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	m.messages = messages
	if len(m.messages) > 0 {
		m.currentMsgID = m.messages[len(m.messages)-1].ID
	}
	delete(m.cachedContent, m.currentMsgID)
	m.rendering = true
	return func() tea.Msg {
		m.renderView()
		return renderFinishedMsg{}
	}
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		m.viewport.KeyMap.PageDown,
		m.viewport.KeyMap.PageUp,
		m.viewport.KeyMap.HalfPageUp,
		m.viewport.KeyMap.HalfPageDown,
	}
}

func NewMessagesCmp(app *app.App) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Pulse
	vp := viewport.New(0, 0)
	attachmets := viewport.New(0, 0)
	vp.KeyMap.PageUp = messageKeys.PageUp
	vp.KeyMap.PageDown = messageKeys.PageDown
	vp.KeyMap.HalfPageUp = messageKeys.HalfPageUp
	vp.KeyMap.HalfPageDown = messageKeys.HalfPageDown
	return &messagesCmp{
		app:           app,
		cachedContent: make(map[string]cacheItem),
		viewport:      vp,
		spinner:       s,
		attachments:   attachmets,
	}
}
