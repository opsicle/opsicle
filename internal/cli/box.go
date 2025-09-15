package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	defaultBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Padding(1, 2)
)

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
