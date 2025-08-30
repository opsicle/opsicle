package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const (
	boxButtonOkay = iota
	boxButtonCancel
)

type boxModel struct {
	IsFullScreen bool
	Message      string
	Width        int
	Height       int

	cursor int // which button is selected
	choice int
}

func (m boxModel) GetChoice() int {
	return m.choice
}

func (m boxModel) Init() tea.Cmd {
	return nil
}

func (m *boxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter":
			if m.cursor == 0 {
				m.choice = boxButtonOkay
			} else {
				m.choice = boxButtonCancel
			}
			return m, tea.Quit
		case "q", "ctrl+c":
			m.choice = boxButtonCancel
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

func (m boxModel) View() string {
	// style for the warning box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("9")). // red
		Padding(1, 2).
		Align(lipgloss.Center)

	btnStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("7")).
		Padding(0, 2)

	btnSelectedNormal := btnStyle.
		BorderStyle(lipgloss.ThickBorder())

	btnSelectedWarning := btnStyle.
		Background(lipgloss.Color("9")).
		BorderStyle(lipgloss.ThickBorder())

	okButton := btnStyle.Render(" OK ")
	cancelButton := btnStyle.Render(" Cancel ")
	if m.cursor == 0 {
		okButton = btnSelectedWarning.Render(" OK ")
	} else {
		cancelButton = btnSelectedNormal.Render(" Cancel ")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, okButton, "  ", cancelButton)

	warningSignStyle := lipgloss.NewStyle().Bold(true)

	box := boxStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n\n",
		warningSignStyle.Render("!!! WARNING !!!"),
		m.Message,
	) + buttons)

	if m.Width == 0 && m.Height == 0 {
		_ = tea.WindowSize()
	}

	if m.IsFullScreen {
		return lipgloss.Place(
			m.Width,
			m.Height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)
	}
	return box + "\n"
}

func ShowWarningWithConfirmation(message string, isFullScreen bool) error {
	warningModel := boxModel{
		IsFullScreen: isFullScreen,
		Message:      message,
		Width:        30,
		Height:       50,
		cursor:       1,
	}
	teaOpts := []tea.ProgramOption{}
	if isFullScreen {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}
	warning := tea.NewProgram(&warningModel, teaOpts...)
	_, err := warning.Run()
	if err != nil {
		return err
	}
	if warningModel.GetChoice() == boxButtonCancel {
		return fmt.Errorf("user cancelled")
	}
	return nil
}

func GetBoxedErrorMessage(message string) string {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 {
		width = 72
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("9")). // red
		Padding(1, 2).
		Width(width - 3).
		Align(lipgloss.Left)
	return boxStyle.Render(
		fmt.Sprintf(
			"%s\n\n%s",
			lipgloss.NewStyle().Bold(true).Render("🔴 ERROR"),
			message,
		),
	)
}
