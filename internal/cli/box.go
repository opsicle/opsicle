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

var (
	defaultBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 2)
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
	OnlyOk          bool

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

	enabledButtons := []string{okButton, "  "}
	if !m.OnlyOk {
		enabledButtons = append(enabledButtons, cancelButton)
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, enabledButtons...)

	titleStyle := lipgloss.NewStyle().Bold(true)

	box := boxStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n\n",
		titleStyle.Render(m.Title),
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

func printBoxedMessage(color AnsiColor, header, message string) {
	width, _, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 || width > 72 {
		width = 72
	}
	header = lipgloss.NewStyle().Bold(true).Render(header)
	boxStyle :=
		defaultBoxStyle.
			BorderForeground(lipgloss.Color(color)).
			Align(lipgloss.Left).
			Width(width)
	fmt.Println(
		boxStyle.Render(fmt.Sprintf("%s\n\n%s", header, message)),
	)
}

func PrintBoxedErrorMessage(message string) {
	printBoxedMessage(AnsiRed, "🔴 ERROR", message)
}

func PrintBoxedInfoMessage(message string) {
	printBoxedMessage(AnsiBlue, "🔵 INFORMATION", message)
}

func PrintBoxedWarningMessage(message string) {
	printBoxedMessage(AnsiYellow, "🟡 WARNING", message)
}

func PrintBoxedSuccessMessage(message string) {
	printBoxedMessage(AnsiGreen, "✅ SUCCESS", message)
}
