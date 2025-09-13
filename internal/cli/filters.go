package cli

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type FilterOpts struct {
	Title              string
	Items              []FilterItem
	StartWithoutFilter bool
}

type FilterItem struct {
	Label       string
	Description string
	Value       string
}

type filterItem struct {
	FilterItem
}

func (i filterItem) Description() string { return i.FilterItem.Description }
func (i filterItem) Title() string       { return i.FilterItem.Label }
func (i filterItem) FilterValue() string { return i.FilterItem.Label }

func CreateFilter(opts FilterOpts) *FilterModel {
	items := []list.Item{}
	for _, item := range opts.Items {
		items = append(items, list.Item(filterItem{item}))
	}
	listInstance := list.New(items, list.NewDefaultDelegate(), 0, 0)
	if !opts.StartWithoutFilter {
		listInstance.SetFilteringEnabled(true)
		listInstance.SetFilterText("")
		listInstance.SetFilterState(list.Filtering)
		listInstance.SetShowFilter(true)
	}
	listInstance.SetShowTitle(true)
	listInstance.SetStatusBarItemName("template", "templates")
	listInstance.SetShowStatusBar(true)
	listInstance.Title = opts.Title
	if opts.Title == "" {
		listInstance.Title = "Select one"
	}
	listInstance.SetShowStatusBar(false)
	return &FilterModel{list: listInstance}
}

type FilterModel struct {
	list list.Model

	selectedItem FilterItem
	isQuitting   bool

	exitCode PromptExitCode
}

func (m FilterModel) GetSelectedItem() FilterItem { return m.selectedItem }
func (m FilterModel) GetExitCode() PromptExitCode { return m.exitCode }
func (m FilterModel) Init() tea.Cmd {
	return nil
}

func (m *FilterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.isQuitting = true
			m.exitCode = PromptCancelled
			return m, tea.Quit
		case "/":
			if m.list.FilterState() != list.Filtering {
				m.list.SetFilterState(list.Filtering)
				m.list.SetShowFilter(true)
				var cmd tea.Cmd
				return m, cmd
			}
		case "down":
			if m.list.FilterState() == list.Filtering {
				m.list.SetFilterState(list.FilterApplied)
				m.list.SetShowFilter(false)
				var cmd tea.Cmd
				return m, cmd
				// m.list.SetFilterState(list.FilterApplied)
			}
		case "up":
			if m.list.FilterState() != list.Filtering && m.list.Cursor() == 0 {
				m.list.SetFilterState(list.Filtering)
				m.list.SetShowFilter(true)
				var cmd tea.Cmd
				return m, cmd
			}
		case "enter":
			if m.list.FilterState() != list.Filtering {
				m.isQuitting = true
				m.selectedItem = m.list.SelectedItem().(filterItem).FilterItem
				m.exitCode = PromptCompleted
				return m, tea.Quit
			} else {
				m.list.SetFilterState(list.FilterApplied)
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m FilterModel) View() string {
	if m.isQuitting {
		return "\n"
	}
	return docStyle.Render(m.list.View())
}
