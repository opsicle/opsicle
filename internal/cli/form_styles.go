package cli

import "github.com/charmbracelet/lipgloss"

var (
	formStyleBold    = lipgloss.NewStyle().Bold(true)
	formStyleDefault = lipgloss.NewStyle()
	formStyleFaded   = lipgloss.NewStyle().Faint(true)

	formStyleError             = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiYellow))
	formStyleInput             = formStyleDefault
	formStyleInputFocused      = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiBlue))
	formStyleInputError        = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiRed))
	formStyleInputDescription  = formStyleFaded
	formStyleInputLabel        = formStyleDefault
	formStyleInputLabelFocused = formStyleBold
	formStylePlaceholder       = formStyleFaded
	formStyleTitle             = formStyleBold
)
