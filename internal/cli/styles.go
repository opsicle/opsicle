package cli

import "github.com/charmbracelet/lipgloss"

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleDefault = lipgloss.NewStyle()
	styleFaded   = lipgloss.NewStyle().Faint(true)

	styleError             = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiYellow))
	styleInput             = styleDefault
	styleInputFocused      = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiBlue))
	styleInputError        = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiRed))
	styleInputDescription  = styleFaded
	styleInputLabel        = styleDefault
	styleInputLabelFocused = styleBold
	stylePlaceholder       = styleFaded
	styleTitle             = styleBold
)
