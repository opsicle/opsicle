package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectorOpts struct {
	Choices []SelectorChoice
}

type SelectorChoice struct {
	Description string
	Label       string
	Value       string
}

func (sc *SelectorChoice) Render(isSelected bool) string {
	if isSelected {
		return fmt.Sprintf("%s %s",
			focusedStyle.Render(fmt.Sprintf("> %s", sc.Label)),
			blurredStyle.Render(fmt.Sprintf("(%s)", sc.Description)),
		)
	}
	return fmt.Sprintf("  %s", blurredStyle.Render(sc.Label))
}

func CreateSelector(opts SelectorOpts) *SelectorModel {
	return &SelectorModel{
		choices:  opts.Choices,
		selected: -1,
	}
}

type SelectorModel struct {
	choices []SelectorChoice

	selected   int
	cursor     int
	isQuitting bool

	exitCode PromptExitCode
}

func (m SelectorModel) GetExitCode() PromptExitCode {
	return m.exitCode
}

func (m SelectorModel) GetDescription() string {
	return m.choices[m.selected].Description
}

func (m SelectorModel) GetLabel() string {
	return m.choices[m.selected].Label
}

func (m SelectorModel) GetValue() string {
	return m.choices[m.selected].Value
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m *SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.isQuitting = true
			m.exitCode = PromptCancelled
			return m, tea.Quit

		case "up", "w", "p":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "s", "n":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter":
			m.selected = m.cursor
			m.isQuitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectorModel) View() string {
	var b strings.Builder

	if len(m.choices) > 0 {
		for i, choice := range m.choices {
			fmt.Fprintf(&b, "%s\n", choice.Render(m.cursor == i))
		}
	}

	if !m.isQuitting {
		fmt.Fprintf(&b, "%s", blurredStyle.Render("\nUse [up]/w/p and [down]/a/n to move, enter to select, q to quit.\n"))
	}

	fmt.Fprintf(&b, "\n")

	return b.String()
}
