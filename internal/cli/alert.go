package cli

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type ShowAlertOpts struct {
	ColorForeground AnsiColor
	ColorAccent     AnsiColor

	Title        string
	Message      string
	IsFullScreen bool

	ConfirmLabel     string
	CancelLabel      string
	IsConfirmDefault bool
}

func ShowAlert(opts ShowAlertOpts) error {
	var height, width int
	if !opts.IsFullScreen {
		width, height, _ = term.GetSize(int(os.Stdout.Fd()))
		if width > 72 {
			width = 72
		}
	}
	accentColor := opts.ColorAccent
	if accentColor == "" {
		accentColor = AnsiOrange
	}
	foregroundColor := opts.ColorForeground
	if foregroundColor == "" {
		foregroundColor = AnsiWhite
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
		Color:           accentColor,
		ConfirmLabel:    confirmLabel,
		ForegroundColor: foregroundColor,
		Height:          height,
		IsFullScreen:    opts.IsFullScreen,
		Message:         opts.Message,
		Title:           opts.Title,
		Width:           width,
		OnlyOk:          true,

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
