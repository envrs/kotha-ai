package diff

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kothagpt/kotha/internal/diff"
	"github.com/kothagpt/kotha/internal/tui/components/chat"
	"github.com/kothagpt/kotha/internal/tui/layout"
	"github.com/kothagpt/kotha/internal/tui/theme"
	"github.com/kothagpt/kotha/internal/tui/util"
)

// FileStatus represents the review status of a file
type FileStatus int

const (
	FileStatusPending FileStatus = iota
	FileStatusReviewing
	FileStatusComplete
	FileStatusSkipped
)

// DiffFile represents a file in the diff review
type DiffFile struct {
	Path        string
	OldPath     string // For renames
	NewPath     string
	Diff        string
	Status      FileStatus
	LineCount   int
	Additions   int
	Removals    int
	IsBinary    bool
	ImageData   []byte // For image previews
	ImageFormat string // png, jpeg, webp, gif
	Expanded    bool   // Whether the diff body is expanded
}

// DiffSource represents the source of the diff (working tree, main branch, etc.)
type DiffSource int

const (
	DiffSourceWorkingTree DiffSource = iota
	DiffSourceMainBranch
	DiffSourceStaged
	DiffSourceCustom
)

func (ds DiffSource) String() string {
	switch ds {
	case DiffSourceWorkingTree:
		return "Working tree"
	case DiffSourceMainBranch:
		return "Main branch"
	case DiffSourceStaged:
		return "Staged"
	case DiffSourceCustom:
		return "Custom"
	}
	return "Unknown"
}

// Mode represents the diff review mode
type Mode int

const (
	ModeAllFiles Mode = iota // Show all files with collapsible diffs
	ModeSingleFile           // Show single file, advance on complete
)

// KeyMap defines key bindings for diff review
type DiffKeyMap struct {
	// Scrolling
	ScrollDown     key.Binding
	ScrollUp       key.Binding
	ScrollPageDown key.Binding
	ScrollPageUp   key.Binding
	ScrollHalfDown key.Binding
	ScrollHalfUp   key.Binding
	ScrollTop      key.Binding
	ScrollBottom   key.Binding

	// File navigation
	NextFile    key.Binding
	PrevFile    key.Binding
	AltNextFile key.Binding
	AltPrevFile key.Binding

	// File actions
	ToggleExpand   key.Binding
	MarkComplete   key.Binding
	MarkIncomplete key.Binding
	ToggleStatus   key.Binding

	// Mode
	ToggleMode key.Binding

	// Diff source
	ChangeSource key.Binding

	// Help
	Help key.Binding

	// Image
	EnlargeImage key.Binding

	// Close/back
	Close key.Binding
}

var defaultDiffKeyMap = DiffKeyMap{
	ScrollDown:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "scroll down")),
	ScrollUp:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "scroll up")),
	ScrollPageDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/pgdn", "page down")),
	ScrollPageUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u/pgup", "page up")),
	ScrollHalfDown: key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "half page down")),
	ScrollHalfUp:   key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "half page up")),
	ScrollTop:      key.NewBinding(key.WithKeys("gg", "home"), key.WithHelp("gg/home", "top")),
	ScrollBottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G/end", "bottom")),

	NextFile:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next file")),
	PrevFile:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev file")),
	AltNextFile:  key.NewBinding(key.WithKeys("alt+down"), key.WithHelp("alt+↓", "next file (alt)")),
	AltPrevFile:  key.NewBinding(key.WithKeys("alt+up"), key.WithHelp("alt+↑", "prev file (alt)")),

	ToggleExpand:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "toggle expand/collapse")),
	MarkComplete:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "mark complete")),
	MarkIncomplete: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "mark incomplete")),
	ToggleStatus:   key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle complete/incomplete")),

	ToggleMode: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle all/single mode")),

	ChangeSource: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "change diff source")),

	Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "show help")),

	EnlargeImage: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "enlarge image")),

	Close: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "close")),
}

// DiffReviewModel is the main model for the diff review page
type DiffReviewModel struct {
	width, height int

	// Data
	files         []*DiffFile
	currentFile   int
	source        DiffSource
	mode          Mode
	sidebarWidth  int
	showSidebar   bool
	showHelp      bool

	// Scrolling
	viewport     viewport.Model // The main diff viewport
	scrollOffset int

	// Image handling
	imageCache     map[string]*image.Image
	imageLoading   map[string]bool
	imageLoadMutex sync.Mutex

	// State
	hoveredFileIdx   int    // For hover effects

	// Key bindings
	keys DiffKeyMap

	// Styles
	styles DiffStyles

	// Callbacks
	onClose         func()
	onSourceChange  func(DiffSource)
	onFileStatusChange func(string, FileStatus)
}

// DiffStyles holds the styling for the diff review
type DiffStyles struct {
	Sidebar         lipgloss.Style
	SidebarHeader   lipgloss.Style
	SidebarFooter   lipgloss.Style
	SourceLabel     lipgloss.Style
	FileList        lipgloss.Style
	FileItem        lipgloss.Style
	FileItemActive  lipgloss.Style
	FileItemHover   lipgloss.Style
	FileStatus      map[FileStatus]lipgloss.Style
	DiffViewport    lipgloss.Style
	FileHeader      lipgloss.Style
	FileHeaderCollapsed lipgloss.Style
	FileHeaderExpanded  lipgloss.Style
	DiffContent     lipgloss.Style
	DiffLineRemoved lipgloss.Style
	DiffLineAdded   lipgloss.Style
	DiffLineContext lipgloss.Style
	LineNumber      lipgloss.Style
	Separator       lipgloss.Style
	HelpHint        lipgloss.Style
	HelpOverlay     lipgloss.Style
	ImagePreview    lipgloss.Style
	ImageLoading    lipgloss.Style
	ImageError      lipgloss.Style
	StatusComplete  lipgloss.Style
	StatusPending   lipgloss.Style
	StatusReviewing lipgloss.Style
	StatusSkipped   lipgloss.Style
	ModeIndicator   lipgloss.Style
	SourceIndicator lipgloss.Style
}

// NewDiffStyles creates the default styles
func NewDiffStyles() DiffStyles {
	t := theme.CurrentTheme()

	return DiffStyles{
		Sidebar: lipgloss.NewStyle().
			Width(30).
			Background(t.Background()).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(t.BorderNormal()).
			Padding(0, 1),

		SidebarHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text()).
			Padding(1, 0).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(t.BorderNormal()),

		SidebarFooter: lipgloss.NewStyle().
			Foreground(t.TextMuted()).
			Padding(1, 0).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(t.BorderNormal()),

		SourceLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary()).
			Padding(0, 1),

		FileList: lipgloss.NewStyle().
			Padding(0, 1),

		FileItem: lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, -1),

		FileItemActive: lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, -1).
			Background(t.BackgroundSecondary()).
			Foreground(t.Text()),

		FileItemHover: lipgloss.NewStyle().
			Padding(0, 1).
			Margin(0, -1).
			Background(t.BackgroundSecondary()).
			Foreground(t.Text()).
			Underline(true),

		FileStatus: map[FileStatus]lipgloss.Style{
			FileStatusPending:   lipgloss.NewStyle().Foreground(t.TextMuted()),
			FileStatusReviewing: lipgloss.NewStyle().Foreground(t.Primary()).Bold(true),
			FileStatusComplete:  lipgloss.NewStyle().Foreground(t.Success()).Bold(true),
			FileStatusSkipped:   lipgloss.NewStyle().Foreground(t.TextMuted()).Italic(true),
		},

		DiffViewport: lipgloss.NewStyle().
			Background(t.Background()).
			Padding(1, 2),

		FileHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text()).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(t.BorderNormal()),

		FileHeaderCollapsed: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Text()).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(t.BorderNormal()).
			Background(t.BackgroundSecondary()),

		FileHeaderExpanded: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Primary()).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(t.Primary()).
			Background(t.BackgroundSecondary()),

		DiffContent: lipgloss.NewStyle().
			Background(t.Background()).
			Padding(0, 1),

		DiffLineRemoved: lipgloss.NewStyle().
			Background(t.DiffRemovedBg()).
			Foreground(t.DiffRemoved()),

		DiffLineAdded: lipgloss.NewStyle().
			Background(t.DiffAddedBg()).
			Foreground(t.DiffAdded()),

		DiffLineContext: lipgloss.NewStyle().
			Background(t.DiffContextBg()).
			Foreground(t.TextMuted()),

		LineNumber: lipgloss.NewStyle().
			Foreground(t.TextMuted()).
			Width(6).
			Align(lipgloss.Right),

		Separator: lipgloss.NewStyle().
			Foreground(t.BorderNormal()).
			Height(0).
			MaxHeight(1),

		HelpHint: lipgloss.NewStyle().
			Foreground(t.TextMuted()).
			Italic(true).
			Padding(0, 1),

		HelpOverlay: lipgloss.NewStyle().
			Background(t.Background()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocused()).
			Padding(1, 2).
			Width(60),

		ImagePreview: lipgloss.NewStyle().
			Background(t.Background()).
			Border(lipgloss.NormalBorder()).
			BorderForeground(t.BorderNormal()).
			Padding(1),

		ImageLoading: lipgloss.NewStyle().
			Foreground(t.TextMuted()).
			Italic(true).
			Padding(1),

		ImageError: lipgloss.NewStyle().
			Foreground(t.Error()).
			Padding(1),

		StatusComplete:  lipgloss.NewStyle().Foreground(t.Success()).Bold(true),
		StatusPending:   lipgloss.NewStyle().Foreground(t.TextMuted()),
		StatusReviewing: lipgloss.NewStyle().Foreground(t.Primary()).Bold(true),
		StatusSkipped:   lipgloss.NewStyle().Foreground(t.TextMuted()).Italic(true),

		ModeIndicator: lipgloss.NewStyle().
			Foreground(t.Primary()).
			Bold(true).
			Padding(0, 1),

		SourceIndicator: lipgloss.NewStyle().
			Foreground(t.Primary()).
			Bold(true).
			Padding(0, 1),
	}
}

// NewDiffReviewModel creates a new diff review model
func NewDiffReviewModel(files []*DiffFile, source DiffSource, mode Mode) *DiffReviewModel {
	m := &DiffReviewModel{
		files:         files,
		source:        source,
		mode:          mode,
		sidebarWidth:  30,
		showSidebar:   true,
		viewport:      viewport.New(80, 24),
		imageCache:    make(map[string]*image.Image),
		imageLoading:  make(map[string]bool),
		keys:          defaultDiffKeyMap,
		styles:        NewDiffStyles(),
	}

	// Set initial current file to first pending/reviewing file
	for i, f := range files {
		if f.Status == FileStatusPending || f.Status == FileStatusReviewing {
			m.currentFile = i
			f.Status = FileStatusReviewing
			break
		}
	}

	return m
}

// Init initializes the model
func (m *DiffReviewModel) Init() tea.Cmd {
	return tea.Batch(
		m.startImageLoading(),
	)
}

// startImageLoading starts loading images for image files
func (m *DiffReviewModel) startImageLoading() tea.Cmd {
	var cmds []tea.Cmd
	for _, f := range m.files {
		if f.IsBinary && (f.ImageFormat == "png" || f.ImageFormat == "jpeg" || f.ImageFormat == "webp" || f.ImageFormat == "gif") {
			cmds = append(cmds, m.loadImage(f.Path))
		}
	}
	return tea.Batch(cmds...)
}

// loadImage loads an image asynchronously
func (m *DiffReviewModel) loadImage(path string) tea.Cmd {
	m.imageLoadMutex.Lock()
	if m.imageLoading[path] {
		m.imageLoadMutex.Unlock()
		return nil
	}
	m.imageLoading[path] = true
	m.imageLoadMutex.Unlock()

	return func() tea.Msg {
		// Read image file
		data, err := os.ReadFile(path)
		if err != nil {
			return imageLoadResult{path: path, err: err}
		}

		img, format, err := image.Decode(strings.NewReader(string(data)))
		if err != nil {
			return imageLoadResult{path: path, err: err}
		}

		m.imageLoadMutex.Lock()
		m.imageCache[path] = &img
		m.imageLoading[path] = false
		m.imageLoadMutex.Unlock()

		return imageLoadResult{path: path, img: img, format: format}
	}
}

// imageLoadResult is the result of loading an image
type imageLoadResult struct {
	path   string
	img    image.Image
	format string
	err    error
}

// Update handles messages
func (m *DiffReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateViewportSize()

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case imageLoadResult:
		// Image loaded
		return m, m.handleImageLoad(msg)
	}

	return m, tea.Batch(cmds...)
}

// handleImageLoad handles image load results
func (m *DiffReviewModel) handleImageLoad(msg imageLoadResult) tea.Cmd {
	if msg.err != nil {
		return util.ReportWarn(fmt.Sprintf("Failed to load image %s: %v", msg.path, msg.err))
	}
	// Trigger re-render
	return nil
}

// updateViewportSize updates the viewport dimensions
func (m *DiffReviewModel) updateViewportSize() {
	// Viewport takes remaining width after sidebar
	diffWidth := m.width - m.sidebarWidth
	if diffWidth < 10 {
		diffWidth = 10
	}
	m.viewport.Width = diffWidth
	m.viewport.Height = m.height
}

// handleKey handles keyboard input
func (m *DiffReviewModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.keys.ScrollDown):
		m.scrollOffset++
		m.viewport.ScrollDown(1)
		return m, nil

	case key.Matches(msg, m.keys.ScrollUp):
		m.scrollOffset = max(0, m.scrollOffset-1)
		m.viewport.ScrollUp(1)
		return m, nil

	case key.Matches(msg, m.keys.ScrollPageDown):
		m.viewport.ScrollDown(m.height / 2)
		return m, nil

	case key.Matches(msg, m.keys.ScrollPageUp):
		m.viewport.ScrollUp(m.height / 2)
		return m, nil

	case key.Matches(msg, m.keys.ScrollHalfDown):
		m.viewport.ScrollDown(m.height / 4)
		return m, nil

	case key.Matches(msg, m.keys.ScrollHalfUp):
		m.viewport.ScrollUp(m.height / 4)
		return m, nil

	case key.Matches(msg, m.keys.ScrollTop):
		m.scrollOffset = 0
		m.viewport.GotoTop()
		return m, nil

	case key.Matches(msg, m.keys.ScrollBottom):
		m.scrollOffset = m.viewport.Height // approximate
		m.viewport.GotoBottom()
		return m, nil

	case key.Matches(msg, m.keys.NextFile):
		return m.nextFile()

	case key.Matches(msg, m.keys.PrevFile):
		return m.prevFile()

	case key.Matches(msg, m.keys.AltNextFile):
		return m.nextFile()

	case key.Matches(msg, m.keys.AltPrevFile):
		return m.prevFile()

	case key.Matches(msg, m.keys.ToggleExpand):
		return m.toggleExpand()

	case key.Matches(msg, m.keys.MarkComplete):
		return m.markCurrentFile(FileStatusComplete)

	case key.Matches(msg, m.keys.MarkIncomplete):
		return m.markCurrentFile(FileStatusPending)

	case key.Matches(msg, m.keys.ToggleStatus):
		return m.toggleStatus()

	case key.Matches(msg, m.keys.ToggleMode):
		return m.toggleMode()

	case key.Matches(msg, m.keys.ChangeSource):
		if m.onSourceChange != nil {
			nextSource := (m.source + 1) % 4
			m.onSourceChange(nextSource)
		}
		return m, util.ReportInfo(fmt.Sprintf("Changed diff source to %s", (m.source+1)%4))

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.EnlargeImage):
		return m.enlargeCurrentImage()

	case key.Matches(msg, m.keys.Close):
		if m.onClose != nil {
			m.onClose()
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
		// Check for gg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// handleMouse handles mouse events
func (m *DiffReviewModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Check if click is in sidebar
			if msg.X < m.sidebarWidth {
				// Click in sidebar - find file
				fileIdx := m.sidebarFileAt(msg.Y)
				if fileIdx >= 0 && fileIdx < len(m.files) {
					m.currentFile = fileIdx
					m.files[m.currentFile].Status = FileStatusReviewing
				}
			} else {
				// Click in diff area
				if msg.Button == tea.MouseButtonRight {
					// Right-click - mark complete/incomplete
					if m.currentFile >= 0 && m.currentFile < len(m.files) {
						f := m.files[m.currentFile]
						if f.Status == FileStatusComplete {
							m.markCurrentFile(FileStatusPending)
						} else {
							m.markCurrentFile(FileStatusComplete)
						}
					}
				}
			}
		}
	case tea.MouseActionMotion:
		if msg.X < m.sidebarWidth {
			m.hoveredFileIdx = m.sidebarFileAt(msg.Y)
		} else {
			m.hoveredFileIdx = -1
		}
	}

	return m, tea.Batch(cmds...)
}

// sidebarFileAt returns the file index at the given Y position in sidebar
func (m *DiffReviewModel) sidebarFileAt(y int) int {
	// Simplified - in reality would need to account for header/footer
	itemHeight := 2 // Approximate height per file item
	idx := (y - 3) / itemHeight // Account for header
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.files) {
		idx = len(m.files) - 1
	}
	return idx
}

// nextFile moves to the next file
func (m *DiffReviewModel) nextFile() (tea.Model, tea.Cmd) {
	if m.currentFile < len(m.files)-1 {
		m.currentFile++
		m.scrollOffset = 0
		m.files[m.currentFile].Status = FileStatusReviewing
		return m, util.ReportInfo(fmt.Sprintf("File %d/%d: %s", m.currentFile+1, len(m.files), m.files[m.currentFile].Path))
	}
	return m, util.ReportInfo("Already at last file")
}

// prevFile moves to the previous file
func (m *DiffReviewModel) prevFile() (tea.Model, tea.Cmd) {
	if m.currentFile > 0 {
		m.currentFile--
		m.scrollOffset = 0
		m.files[m.currentFile].Status = FileStatusReviewing
		return m, util.ReportInfo(fmt.Sprintf("File %d/%d: %s", m.currentFile+1, len(m.files), m.files[m.currentFile].Path))
	}
	return m, util.ReportInfo("Already at first file")
}

// toggleExpand toggles the expanded state of the current file
func (m *DiffReviewModel) toggleExpand() (tea.Model, tea.Cmd) {
	if m.currentFile >= 0 && m.currentFile < len(m.files) {
		f := m.files[m.currentFile]
		f.Expanded = !f.Expanded
		if f.Expanded {
			return m, util.ReportInfo(fmt.Sprintf("Expanded %s", filepath.Base(f.Path)))
		}
		return m, util.ReportInfo(fmt.Sprintf("Collapsed %s", filepath.Base(f.Path)))
	}
	return m, nil
}

// markCurrentFile sets the status of the current file
func (m *DiffReviewModel) markCurrentFile(status FileStatus) (tea.Model, tea.Cmd) {
	if m.currentFile >= 0 && m.currentFile < len(m.files) {
		m.files[m.currentFile].Status = status
		if m.onFileStatusChange != nil {
			m.onFileStatusChange(m.files[m.currentFile].Path, status)
		}
		return m, util.ReportInfo(fmt.Sprintf("Marked %s as %s", filepath.Base(m.files[m.currentFile].Path), statusString(status)))
	}
	return m, nil
}

// toggleStatus toggles between complete and pending
func (m *DiffReviewModel) toggleStatus() (tea.Model, tea.Cmd) {
	if m.currentFile >= 0 && m.currentFile < len(m.files) {
		f := m.files[m.currentFile]
		if f.Status == FileStatusComplete {
			return m.markCurrentFile(FileStatusPending)
		}
		return m.markCurrentFile(FileStatusComplete)
	}
	return m, nil
}

// toggleMode toggles between all-files and single-file mode
func (m *DiffReviewModel) toggleMode() (tea.Model, tea.Cmd) {
	if m.mode == ModeAllFiles {
		m.mode = ModeSingleFile
		return m, util.ReportInfo("Switched to single-file mode")
	}
	m.mode = ModeAllFiles
	return m, util.ReportInfo("Switched to all-files mode")
}

// enlargeCurrentImage opens the current image in the image modal
func (m *DiffReviewModel) enlargeCurrentImage() (tea.Model, tea.Cmd) {
	if m.currentFile >= 0 && m.currentFile < len(m.files) {
		f := m.files[m.currentFile]
		if f.IsBinary && f.ImageData != nil {
			// Open image modal
			return m, util.CmdHandler(chat.ImageEnlargeMsg{Data: f.ImageData, Format: f.ImageFormat})
		}
	}
	return m, util.ReportWarn("No image to enlarge")
}

// refresh refreshes the diff data
// View renders the diff review
func (m *DiffReviewModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Render sidebar
	sidebar := m.renderSidebar()

	// Render diff viewport
	diffView := m.renderDiffViewport()

	// Join sidebar and diff view
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, diffView)

	// Add help overlay if shown
	if m.showHelp {
		helpOverlay := m.renderHelpOverlay()
		mainView = layout.PlaceOverlay(
			m.width/2-lipgloss.Width(helpOverlay)/2,
			m.height/2-lipgloss.Height(helpOverlay)/2,
			helpOverlay,
			mainView,
			true,
		)
	}

	return mainView
}

// renderSidebar renders the sidebar with file list and source selector
func (m *DiffReviewModel) renderSidebar() string {
	// Source selector
	sourceSelector := m.renderSourceSelector()

	// File list
	fileList := m.renderFileList()

	// Help hint
	helpHint := m.styles.HelpHint.Render("? help  d source  m expand  space toggle  n/p next/prev  j/k scroll")

	// Mode indicator
	modeText := "All files"
	if m.mode == ModeSingleFile {
		modeText = "Single file"
	}
	modeIndicator := m.styles.ModeIndicator.Render(fmt.Sprintf("Mode: %s", modeText))

	sidebarContent := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.SidebarHeader.Render("DIFF REVIEW"),
		sourceSelector,
		m.styles.Separator.Render(strings.Repeat("─", m.sidebarWidth-2)),
		fileList,
		m.styles.Separator.Render(strings.Repeat("─", m.sidebarWidth-2)),
		modeIndicator,
		helpHint,
	)

	return m.styles.Sidebar.
		Width(m.sidebarWidth).
		Height(m.height).
		Render(sidebarContent)
}

// renderSourceSelector renders the diff source selector
func (m *DiffReviewModel) renderSourceSelector() string {
	t := theme.CurrentTheme()

	sources := []DiffSource{
		DiffSourceWorkingTree,
		DiffSourceMainBranch,
		DiffSourceStaged,
		DiffSourceCustom,
	}

	var items []string
	for _, s := range sources {
		style := lipgloss.NewStyle().Padding(0, 1)
		if s == m.source {
			style = style.
				Bold(true).
				Foreground(t.Primary()).
				Background(t.BackgroundSecondary()).
				Padding(0, 1)
		}
		items = append(items, style.Render(s.String()))
	}

	return m.styles.SourceLabel.Render("Source:") + "\n" +
		lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderFileList renders the list of files in the sidebar
func (m *DiffReviewModel) renderFileList() string {
	var items []string

	for i, f := range m.files {
		isActive := i == m.currentFile
		isHovered := i == m.hoveredFileIdx

		// Status icon
		statusStyle := m.styles.FileStatus[f.Status]
		var statusIcon string
		switch f.Status {
		case FileStatusPending:
			statusIcon = "○"
		case FileStatusReviewing:
			statusIcon = "◐"
		case FileStatusComplete:
			statusIcon = "✓"
		case FileStatusSkipped:
			statusIcon = "⊘"
		}

		// File name
		name := filepath.Base(f.Path)
		if f.OldPath != "" && f.OldPath != f.Path {
			name = filepath.Base(f.OldPath) + " → " + name
		}

		// Stats
		stats := fmt.Sprintf("+%d/-%d", f.Additions, f.Removals)

		// Build item
		item := lipgloss.JoinHorizontal(lipgloss.Left,
			statusStyle.Render(statusIcon),
			" ",
			lipgloss.NewStyle().Width(20).Render(name),
			" ",
			lipgloss.NewStyle().Foreground(theme.CurrentTheme().TextMuted()).Render(stats),
		)

		// Apply active/hover styles
		if isActive {
			item = m.styles.FileItemActive.Render(item)
		} else if isHovered {
			item = m.styles.FileItemHover.Render(item)
		} else {
			item = m.styles.FileItem.Render(item)
		}

		// Add collapsed indicator
		if !f.Expanded && len(f.Diff) > 0 {
			item += " " + lipgloss.NewStyle().Foreground(theme.CurrentTheme().TextMuted()).Render("▶")
		}

		items = append(items, item)
	}

	return m.styles.FileList.Render(lipgloss.JoinVertical(lipgloss.Left, items...))
}

// renderDiffViewport renders the diff content area
func (m *DiffReviewModel) renderDiffViewport() string {
	if m.currentFile < 0 || m.currentFile >= len(m.files) {
		m.viewport.SetContent("No file selected")
		return m.styles.DiffViewport.
			Width(m.width - m.sidebarWidth).
			Height(m.height).
			Render(m.viewport.View())
	}

	f := m.files[m.currentFile]

	// File header
	header := m.renderFileHeader(f)

	// Diff content
	var content string
	if f.IsBinary {
		content = m.renderImagePreview(f)
	} else if f.Expanded || m.mode == ModeSingleFile {
		content = m.renderDiffContent(f)
	} else {
		content = m.styles.DiffContent.Render(
			lipgloss.NewStyle().
				Foreground(theme.CurrentTheme().TextMuted()).
				Italic(true).
				Render(fmt.Sprintf("Press m to expand (%d lines, +%d/-%d)", f.LineCount, f.Additions, f.Removals)),
		)
	}

	// Combine header and content
	fullContent := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// Set content in viewport and render
	m.viewport.SetContent(fullContent)
	return m.styles.DiffViewport.
		Width(m.width - m.sidebarWidth).
		Height(m.height).
		Render(m.viewport.View())
}

// renderFileHeader renders the file header with path and stats
func (m *DiffReviewModel) renderFileHeader(f *DiffFile) string {
	t := theme.CurrentTheme()

	// Status indicator
	statusStyle := m.styles.FileStatus[f.Status]
	var statusText string
	switch f.Status {
	case FileStatusPending:
		statusText = "PENDING"
	case FileStatusReviewing:
		statusText = "REVIEWING"
	case FileStatusComplete:
		statusText = "COMPLETE"
	case FileStatusSkipped:
		statusText = "SKIPPED"
	}

	// Path
	var pathText string
	if f.OldPath != "" && f.OldPath != f.Path {
		pathText = fmt.Sprintf("%s → %s", f.OldPath, f.Path)
	} else {
		pathText = f.Path
	}

	// Stats
	statsText := fmt.Sprintf("+%d −%d  %d lines", f.Additions, f.Removals, f.LineCount)

	// Mode indicator
	modeText := ""
	if m.mode == ModeSingleFile {
		modeText = " [single]"
	} else {
		modeText = " [all]"
	}

	// Choose header style based on expanded state
	var headerStyle lipgloss.Style
	if f.Expanded || m.mode == ModeSingleFile {
		headerStyle = m.styles.FileHeaderExpanded
	} else {
		headerStyle = m.styles.FileHeaderCollapsed
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		statusStyle.Render("● "+statusText),
		" ",
		headerStyle.Render(pathText),
		" ",
		lipgloss.NewStyle().Foreground(t.TextMuted()).Render(statsText),
		modeText,
	)

	return m.styles.DiffViewport.Width(m.width - m.sidebarWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.styles.Separator.Render(strings.Repeat("─", m.width-m.sidebarWidth-4)),
			header,
			m.styles.Separator.Render(strings.Repeat("─", m.width-m.sidebarWidth-4)),
		),
	)
}

// renderDiffContent renders the diff content with syntax highlighting
func (m *DiffReviewModel) renderDiffContent(f *DiffFile) string {
	diffResult, err := diff.ParseUnifiedDiff(f.Diff)
	if err != nil {
		return m.styles.DiffContent.Render("Error parsing diff: " + err.Error())
	}

	var sb strings.Builder
	for _, hunk := range diffResult.Hunks {
		// Render hunk header
		hunkHeader := lipgloss.NewStyle().
			Foreground(theme.CurrentTheme().TextMuted()).
			Bold(true).
			Render(hunk.Header)
		sb.WriteString(hunkHeader + "\n")

		// Render lines
		for _, line := range hunk.Lines {
			var lineStyle lipgloss.Style
			var prefix string

			switch line.Kind {
			case diff.LineRemoved:
				lineStyle = m.styles.DiffLineRemoved
				prefix = "-"
			case diff.LineAdded:
				lineStyle = m.styles.DiffLineAdded
				prefix = "+"
			case diff.LineContext:
				lineStyle = m.styles.DiffLineContext
				prefix = " "
			}

			// Line numbers
			lineNum := ""
			if line.OldLineNo > 0 && line.NewLineNo > 0 {
				lineNum = fmt.Sprintf("%6d %6d ", line.OldLineNo, line.NewLineNo)
			} else if line.OldLineNo > 0 {
				lineNum = fmt.Sprintf("%6d      ", line.OldLineNo)
			} else if line.NewLineNo > 0 {
				lineNum = fmt.Sprintf("      %6d ", line.NewLineNo)
			} else {
				lineNum = "            "
			}

			lineNumStyled := m.styles.LineNumber.Render(lineNum)

			// Apply syntax highlighting if available
			content := line.Content
			_ = line.Segments // Intra-line highlighting not implemented yet

			lineContent := lineStyle.Render(prefix + content)
			sb.WriteString(lineNumStyled + lineContent + "\n")
		}
	}

	return m.styles.DiffContent.Render(sb.String())
}

// renderImagePreview renders an image preview
func (m *DiffReviewModel) renderImagePreview(f *DiffFile) string {
	m.imageLoadMutex.Lock()
	img := m.imageCache[f.Path]
	loading := m.imageLoading[f.Path]
	m.imageLoadMutex.Unlock()

	if loading {
		return m.styles.ImageLoading.Render("Loading image preview...")
	}

	if img != nil {
		// Render compact preview
		imgVal := *img
		bounds := imgVal.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		info := fmt.Sprintf("%s  %dx%d  %s", filepath.Base(f.Path), width, height, strings.ToUpper(f.ImageFormat))

		return m.styles.ImagePreview.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Bold(true).Render(info),
				lipgloss.NewStyle().Foreground(theme.CurrentTheme().TextMuted()).Render("Press Enter to enlarge"),
			),
		)
	}

	return m.styles.ImageError.Render("Failed to load image preview")
}

// renderHelpOverlay renders the help overlay
func (m *DiffReviewModel) renderHelpOverlay() string {
	helpText := []string{
		m.styles.HelpHint.Bold(true).Render("DIFF REVIEW SHORTCUTS"),
		"",
		"SCROLLING:",
		"  j / ↓           Scroll down",
		"  k / ↑           Scroll up",
		"  Ctrl+D / PgDn   Page down",
		"  Ctrl+U / PgUp   Page up",
		"  Ctrl+F          Half page down",
		"  Ctrl+B          Half page up",
		"  gg / Home       Top",
		"  G / End         Bottom",
		"",
		"FILE NAVIGATION:",
		"  n / Alt+↓       Next file",
		"  p / Alt+↑       Previous file",
		"  Click file      Select file",
		"",
		"FILE ACTIONS:",
		"  m               Toggle expand/collapse",
		"  Space / c / i   Toggle / mark complete / incomplete",
		"  Right-click     Toggle complete/incomplete",
		"",
		"MODE:",
		"  Tab             Toggle all-files / single-file mode",
		"",
		"DIFF SOURCE:",
		"  d               Change diff source",
		"  Click source    Select source",
		"",
		"IMAGES:",
		"  Enter           Enlarge image",
		"",
		"HELP:",
		"  ?               Toggle this help",
		"  esc / q         Close help / close diff review",
	}

	return m.styles.HelpOverlay.Render(lipgloss.JoinVertical(lipgloss.Left, helpText...))
}

// SetOnClose sets the close callback
func (m *DiffReviewModel) SetOnClose(fn func()) {
	m.onClose = fn
}

// SetOnSourceChange sets the source change callback
func (m *DiffReviewModel) SetOnSourceChange(fn func(DiffSource)) {
	m.onSourceChange = fn
}

// SetOnFileStatusChange sets the file status change callback
func (m *DiffReviewModel) SetOnFileStatusChange(fn func(string, FileStatus)) {
	m.onFileStatusChange = fn
}

// AddFile adds a file to the diff review
func (m *DiffReviewModel) AddFile(f *DiffFile) {
	m.files = append(m.files, f)
}

// SetFiles replaces the file list
func (m *DiffReviewModel) SetFiles(files []*DiffFile) {
	m.files = files
	m.currentFile = 0
	for i, f := range files {
		if f.Status == FileStatusPending || f.Status == FileStatusReviewing {
			m.currentFile = i
			f.Status = FileStatusReviewing
			break
		}
	}
}

// statusString returns the string representation of a FileStatus
func statusString(s FileStatus) string {
	switch s {
	case FileStatusPending:
		return "pending"
	case FileStatusReviewing:
		return "reviewing"
	case FileStatusComplete:
		return "complete"
	case FileStatusSkipped:
		return "skipped"
	}
	return "unknown"
}

// SetSource sets the diff source
func (m *DiffReviewModel) SetSource(source DiffSource) {
	m.source = source
}

// SetMode sets the review mode
func (m *DiffReviewModel) SetMode(mode Mode) {
	m.mode = mode
}