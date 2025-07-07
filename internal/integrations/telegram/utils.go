package telegram

import (
	"fmt"

	"github.com/go-telegram/bot"
)

func FormatInputf(input string, args ...string) string {
	i := []any{}
	for _, arg := range args {
		i = append(i, bot.EscapeMarkdown(arg))
	}
	return fmt.Sprintf(
		input,
		i...,
	)
}
