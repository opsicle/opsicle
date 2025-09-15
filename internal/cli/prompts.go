package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiBlue))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiGray))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
)

type PromptOpts struct {
	Title   string
	Buttons []PromptButton
	Inputs  []PromptInput
}

func CreatePrompt(opts PromptOpts) *PromptModel {
	m := PromptModel{
		buttons:           opts.Buttons,
		inputs:            []textinput.Model{},
		inputPrompts:      []PromptInput{},
		inputIndex:        map[string]int{},
		inputReverseIndex: map[int]string{},
		outputs:           map[string]string{},
	}
	if opts.Title != "" {
		m.title = &opts.Title
	}

	inputLength := 0
	for _, input := range opts.Inputs {
		switch input.Type {
		case PromptPassword:
			fallthrough
		case PromptString:
			value, ok := input.Value.(string)
			if !ok || value == "" {
				m.inputIndex[input.Id] = inputLength
				m.inputReverseIndex[inputLength] = input.Id
				inputLength++
				m.inputPrompts = append(m.inputPrompts, input)
			} else {
				m.outputs[input.Id] = value
			}
		case PromptInteger:
			value, ok := input.Value.(int)
			if !ok || value == 0 {
				m.inputIndex[input.Id] = inputLength
				m.inputReverseIndex[inputLength] = input.Id
				inputLength++
				m.inputPrompts = append(m.inputPrompts, input)
			} else {
				m.outputs[input.Id] = fmt.Sprintf("%v", value)
			}
		}
	}
	m.inputs = make([]textinput.Model, inputLength)

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.Width = 64
		if i == 0 {
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		}
		m.inputPrompts[i].Render(&t)
		m.inputs[i] = t
	}

	return &m
}

type PromptExitCode int

const (
	PromptCompleted PromptExitCode = 0
	PromptCancelled PromptExitCode = 1
)

type PromptModel struct {
	cursorMode cursor.Mode
	focusIndex int

	buttons           []PromptButton
	inputs            []textinput.Model
	inputPrompts      []PromptInput
	inputReverseIndex map[int]string
	inputIndex        map[string]int
	isQuitting        bool
	outputs           map[string]string
	title             *string

	exitCode PromptExitCode
}

func (m PromptModel) GetExitCode() PromptExitCode {
	return m.exitCode
}

func (m PromptModel) GetValue(id string) string {
	return m.outputs[id]
}

func (m PromptModel) getTotalItemsCount() int {
	return len(m.buttons) + len(m.inputs)
}

func (m PromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *PromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.inputs) == 0 {
		m.exitCode = PromptCompleted
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.exitCode = PromptCancelled
			m.isQuitting = true
			return m, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down", "left", "right":
			s := msg.String()

			if s == "enter" && m.focusIndex >= len(m.inputs) {
				// fmt.Printf("focusIndex[%v], len(m.inputs)[%v]\n", m.focusIndex, len(m.inputs))
				m.buttons[m.focusIndex-len(m.inputs)].Handle(m)
				// fmt.Printf("exit code: %v\n", m.exitCode)
				m.isQuitting = true
				return m, tea.Quit
			}

			if s == "left" {
				if m.focusIndex >= len(m.inputs) {
					m.focusIndex--
				} else {
					break
				}
			} else if s == "right" {
				if m.focusIndex >= len(m.inputs) {
					m.focusIndex++
				} else {
					break
				}
			} else if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > m.getTotalItemsCount()-1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = m.getTotalItemsCount() - 1
			}

			// cmds := make([]tea.Cmd, len(m.inputs))
			cmds := make([]tea.Cmd, m.getTotalItemsCount())
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *PromptModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m PromptModel) View() string {
	var b strings.Builder

	if m.title != nil {
		fmt.Fprintf(&b, "%s\n\n", *m.title)
	}

	if len(m.inputs) > 0 {
		for i := range m.inputs {
			b.WriteString(m.inputs[i].View())
			if i < len(m.inputs)-1 {
				b.WriteRune('\n')
			}
		}

		if !m.isQuitting {
			if len(m.buttons) > 0 {
				fmt.Fprintf(&b, "\n\n")
				for i := range m.buttons {
					fmt.Fprintf(&b, "%s\t", m.buttons[i].Render(m.focusIndex == len(m.inputs)+i))
				}
				fmt.Fprintf(&b, "\n")
			}
		} else {
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

type PromptButtonType string

const (
	PromptButtonCancel PromptButtonType = "cancel"
	PromptButtonSubmit PromptButtonType = "submit"
)

type PromptButton struct {
	Id    string
	Label string
	Type  PromptButtonType
}

func (pb *PromptButton) Handle(m *PromptModel) {
	switch pb.Type {
	case PromptButtonCancel:
		m.exitCode = PromptCancelled
	case PromptButtonSubmit:
		for i, input := range m.inputs {
			m.outputs[m.inputReverseIndex[i]] = input.Value()
		}
		m.exitCode = PromptCompleted
	}
}

func (pb *PromptButton) Render(isSelected bool) string {
	if isSelected {
		return focusedStyle.Render(fmt.Sprintf("[ %s ]", pb.Label))
	}
	return fmt.Sprintf("[ %s ]", blurredStyle.Render(pb.Label))
}

type PromptInputType string

const (
	PromptInteger  PromptInputType = "integer"
	PromptString   PromptInputType = "string"
	PromptPassword PromptInputType = "password"
)

type PromptInput struct {
	Id          string
	Type        PromptInputType
	Placeholder string
	Value       any
}

func (pi *PromptInput) Render(t *textinput.Model) {
	switch pi.Type {
	case PromptInteger:
		fallthrough
	case PromptString:
		t.CharLimit = 256
		t.Placeholder = pi.Placeholder
	case PromptPassword:
		t.CharLimit = 256
		t.Placeholder = pi.Placeholder
		t.EchoMode = textinput.EchoPassword
		t.EchoCharacter = '*'
	}
}
