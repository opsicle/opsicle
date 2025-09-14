package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	checkBoxInstructions = "Use up/down arrows to navigate, use space to toggle"
)

type CheckboxItem struct {
	Id          string `json:"id" yaml:"id"`
	Label       string `json:"label" yaml:"label"`
	Checked     bool   `json:"checked" yaml:"checked"`
	Description string `json:"description" yaml:"description"`
	Disabled    bool   `json:"disabled" yaml:"disabled"`
}

type CheckboxModel struct {
	title       string
	itemMap     map[string]CheckboxItem
	items       []CheckboxItem
	cursor      int
	isSubmitted bool
	isCancelled bool
}

type CreateCheckboxesOpts struct {
	Title string
	Items []CheckboxItem
}

func CreateCheckboxes(opts CreateCheckboxesOpts) *CheckboxModel {
	checkboxModel := &CheckboxModel{
		title:   opts.Title,
		items:   opts.Items,
		itemMap: map[string]CheckboxItem{},
	}
	for _, item := range opts.Items {
		checkboxModel.itemMap[item.Id] = item
	}
	return checkboxModel
}

func (m CheckboxModel) GetItemStatus(id string) bool {
	return m.itemMap[id].Checked
}

func (m CheckboxModel) GetItems() []CheckboxItem {
	return m.items
}

func (m CheckboxModel) IsCancelled() bool {
	return m.isCancelled
}

func (m CheckboxModel) Init() tea.Cmd { return nil }

func (m *CheckboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			m.isCancelled = true
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.items) - 1
			}
		case "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case " ":
			if len(m.items) > 0 && !m.items[m.cursor].Disabled && !m.isSubmitted {
				m.items[m.cursor].Checked = !m.items[m.cursor].Checked
			}
		case "enter":
			m.isSubmitted = true
			for _, item := range m.items {
				m.itemMap[item.Id] = item
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m CheckboxModel) View() string {
	var b strings.Builder

	// Title
	fmt.Fprintf(&b, "💬 %s\n  > %s\n\n", m.title, lipgloss.NewStyle().Faint(true).Render(checkBoxInstructions))

	// Checkboxes
	for i, it := range m.items {
		cursor := " "
		if m.cursor == i && !m.isSubmitted {
			cursor = ">"
		}

		box := "[ ]"
		if it.Checked {
			box = "[x]"
		}
		if it.Disabled {
			box = "[-]"
		}

		fmt.Fprintf(&b, "  %s %s %s\n", cursor, box, it.Label)
	}

	if !m.isSubmitted && !m.isCancelled && m.items[m.cursor].Description != "" {
		style := lipgloss.NewStyle().Faint(true)
		fmt.Fprintf(&b, "\n%s\n", style.Render(m.items[m.cursor].Description))
	}

	if m.isCancelled || m.isSubmitted {
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

func selected(items []CheckboxItem) []string {
	var out []string
	for _, it := range items {
		if it.Checked {
			out = append(out, it.Label)
		}
	}
	return out
}
