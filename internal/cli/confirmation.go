package cli

import (
	"errors"
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

var (
	ErrorUserCancelled = errors.New("user_cancelled")
)

var (
	defaultButtonStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 5)
)

type boxModel struct {
	CancelLabel     string
	Color           AnsiColor
	ConfirmLabel    string
	ForegroundColor AnsiColor
	Height          int
	IsFullScreen    bool
	Message         string
	Title           string
	Width           int

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
		if m.IsFullScreen {
			m.Width = msg.Width
			m.Height = msg.Height
		}
	}
	return m, nil
}

func (m boxModel) View() string {
	// style for the warning box
	boxStyle := defaultBoxStyle.BorderForeground(lipgloss.Color(m.Color))

	if m.IsFullScreen {
		boxStyle = boxStyle.Width(m.Width - 2).Align(lipgloss.Center)
	} else {
		boxStyle = boxStyle.Width(m.Width)
	}

	buttonStyle := defaultButtonStyle.BorderForeground(lipgloss.Color(m.ForegroundColor))
	buttonSelectedStyle := buttonStyle.BorderStyle(lipgloss.ThickBorder())
	buttonSelectedNormal := buttonSelectedStyle
	buttonSelectedWarning := buttonSelectedStyle.Background(lipgloss.Color(m.Color))

	okButtonStyle := buttonStyle
	cancelButtonStyle := buttonStyle

	if m.cursor == 0 {
		okButtonStyle = buttonSelectedWarning
	} else {
		cancelButtonStyle = buttonSelectedNormal
	}
	okButton := okButtonStyle.Render(m.ConfirmLabel)
	cancelButton := cancelButtonStyle.Render(m.CancelLabel)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, okButton, "  ", cancelButton)

	warningSignStyle := lipgloss.NewStyle().Bold(true)

	box := boxStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n\n",
		warningSignStyle.Render(m.Title),
		m.Message,
	) + buttons)

	if m.IsFullScreen {
		m.Width, m.Height, _ = term.GetSize(int(os.Stdout.Fd()))
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

type ShowConfirmationOpts struct {
	ColorForeground AnsiColor
	ColorAccent     AnsiColor

	Title        string
	Message      string
	IsFullScreen bool

	ConfirmLabel     string
	CancelLabel      string
	IsConfirmDefault bool
}

func ShowConfirmation(opts ShowConfirmationOpts) error {
	var height, width int
	if !opts.IsFullScreen {
		width, height, _ = term.GetSize(int(os.Stdout.Fd()))
		if width > 72 {
			width = 72
		}
	}
	accentColor := opts.ColorAccent
	if accentColor == "" {
		accentColor = AnsiBlue
	}
	foregroundColor := opts.ColorForeground
	if foregroundColor == "" {
		foregroundColor = AnsiWhite
	}
	cancelLabel := opts.CancelLabel
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}
	confirmLabel := opts.ConfirmLabel
	if confirmLabel == "" {
		confirmLabel = "OK"
	}
	initialCursorPosition := 1
	if opts.IsConfirmDefault {
		initialCursorPosition = 0
	}
	model := boxModel{
		CancelLabel:     cancelLabel,
		Color:           accentColor,
		ConfirmLabel:    confirmLabel,
		ForegroundColor: foregroundColor,
		Height:          height,
		IsFullScreen:    opts.IsFullScreen,
		Message:         opts.Message,
		Title:           opts.Title,
		Width:           width,

		cursor: initialCursorPosition,
	}
	teaOpts := []tea.ProgramOption{}
	if opts.IsFullScreen {
		teaOpts = append(teaOpts, tea.WithAltScreen())
	}
	program := tea.NewProgram(&model, teaOpts...)
	_, err := program.Run()
	if err != nil {
		return err
	}
	if model.GetChoice() == boxButtonCancel {
		return ErrorUserCancelled
	}
	return nil
}

func ShowWarningWithConfirmation(message string, isFullScreen bool) error {
	return ShowConfirmation(ShowConfirmationOpts{
		ColorAccent:     AnsiRed,
		ColorForeground: AnsiWhite,
		Title:           "🔴🔴🔴 WARNING 🔴🔴🔴",
		Message:         message,
		IsFullScreen:    isFullScreen,
		ConfirmLabel:    "OK",
		CancelLabel:     "Cancel",
	})
}
