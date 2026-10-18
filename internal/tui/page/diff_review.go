package page

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kothagpt/kotha/internal/app"
	"github.com/kothagpt/kotha/internal/tui/components/diff"
	"github.com/kothagpt/kotha/internal/tui/layout"
)

var DiffReviewPage PageID = "diff_review"

// DiffReviewPageModel wraps the diff review component for the page system
type DiffReviewPageModel struct {
	diffModel *diff.DiffReviewModel
	layout    layout.Container
	app       *app.App
}

func NewDiffReviewPage(app *app.App) tea.Model {
	diffModel := diff.NewDiffReviewModel(nil, diff.DiffSourceWorkingTree, diff.ModeAllFiles)
	diffModel.SetOnClose(func() {
		// Return to chat page when closed
	})

	return &DiffReviewPageModel{
		diffModel: diffModel,
		app:       app,
		layout:    layout.NewContainer(diffModel, layout.WithPadding(0, 0, 0, 0)),
	}
}

func (p *DiffReviewPageModel) Init() tea.Cmd {
	return p.diffModel.Init()
}

func (p *DiffReviewPageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update diff model
	updatedDiff, cmd := p.diffModel.Update(msg)
	p.diffModel = updatedDiff.(*diff.DiffReviewModel)
	cmds = append(cmds, cmd)

	// Update layout
	updatedLayout, cmd := p.layout.Update(msg)
	p.layout = updatedLayout.(layout.Container)
	cmds = append(cmds, cmd)

	return p, tea.Batch(cmds...)
}

func (p *DiffReviewPageModel) View() string {
	return p.layout.View()
}

func (p *DiffReviewPageModel) SetSize(width, height int) tea.Cmd {
	return p.layout.SetSize(width, height)
}

func (p *DiffReviewPageModel) GetSize() (int, int) {
	return p.layout.GetSize()
}