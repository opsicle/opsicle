package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
)

func getFormButtons() FormButtons {
	return FormButtons{
		{
			Label: "Submit",
			Type:  FormButtonSubmit,
		},
		{
			Label: "Cancel",
			Type:  FormButtonCancel,
		},
	}
}

const (
	FormFieldBoolean = "boolean"
	FormFieldFloat   = "float"
	FormFieldInteger = "integer"
	FormFieldString  = "string"

	FormButtonCancel FormButtonType = "cancel"
	FormButtonSubmit FormButtonType = "submit"
)

type FormOpts struct {
	Description string
	Fields      FormFields
	Title       string
}

func CreateForm(opts FormOpts) *FormModel {
	logrus.Infof("createform started")
	var inputs FormFields
	for i, field := range opts.Fields {
		inputModel := textinput.New()
		inputModel.Width = 64
		if i == 0 {
			inputModel.Focus()
		}
		if field.DefaultValue != nil {
			inputModel.SetValue(fmt.Sprintf("%v", field.DefaultValue))
			switch field.Type {
			case FormFieldBoolean:
				inputModel.Validate = func(s string) error {
					if field.IsRequired && s == "" {
						return fmt.Errorf("this field is required")
					}
					val := strings.ToLower(s)
					switch val {
					case "1", "true", "0", "false":
						return nil
					}
					return fmt.Errorf("this field should be a boolean")
				}
			case FormFieldInteger:
				inputModel.Validate = func(s string) error {
					if field.IsRequired && s == "" {
						return fmt.Errorf("this field is required")
					} else if _, err := strconv.Atoi(s); err != nil {
						return fmt.Errorf("this field should be an integer: %w", err)
					}
					return nil
				}
			case FormFieldFloat:
				inputModel.Validate = func(s string) error {
					if field.IsRequired && s == "" {
						return fmt.Errorf("this field is required")
					} else if _, err := strconv.ParseFloat(s, 64); err != nil {
						return fmt.Errorf("this field should be a floating point number: %w", err)
					}
					return nil
				}
			case FormFieldString:
				fallthrough
			default:
				inputModel.Validate = func(s string) error {
					if field.IsRequired && s == "" {
						return fmt.Errorf("this field is required")
					}
					return nil
				}
			}
		}
		inputModel.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
		inputs = append(inputs, FormField{
			Id:           field.Id,
			Description:  field.Description,
			Label:        field.Label,
			DefaultValue: field.DefaultValue,
			Value:        field.Value,
			Model:        inputModel,
		})
	}
	return &FormModel{
		description: opts.Description,
		inputs:      inputs,
		title:       opts.Title,
		buttons:     getFormButtons(),
	}
}

type FormModel struct {
	buttons     []FormButton
	description string
	err         error
	exitCode    error
	focused     int
	inputs      []FormField
	isExitting  bool
	title       string
}

type FormButtons []FormButton

type FormButton struct {
	Label string
	Type  FormButtonType
}

type FormButtonType string

func (fb *FormButton) Render(isSelected bool) string {
	if isSelected {
		return focusedStyle.Render(fmt.Sprintf("[ %s ]", fb.Label))
	}
	return fmt.Sprintf("[ %s ]", blurredStyle.Render(fb.Label))
}

type FormFields []FormField

type FormFieldType string

type FormField struct {
	Id           string
	Label        string
	Description  string
	Value        any
	DefaultValue any
	Type         FormFieldType
	IsRequired   bool
	textinput.Model
}

func (m *FormModel) GetData() map[string]any {
	output := map[string]any{}
	for _, input := range m.inputs {
		output[input.Id] = input.Value
	}
	return output
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focused < len(m.inputs) {
				m.nextInput()
			} else {
				switch m.buttons[m.focused-len(m.inputs)].Type {
				case FormButtonCancel:
					m.isExitting = true
					m.exitCode = ErrorUserCancelled
					return m, tea.Quit
				case FormButtonSubmit:
					hasErrors := false
					firstErrorIndex := len(m.inputs)
					for i, input := range m.inputs {
						if err := input.Validate(input.Model.Value()); err != nil {
							if i < firstErrorIndex {
								firstErrorIndex = i
							}
							m.inputs[i].Err = err
							hasErrors = true
						}
						m.inputs[i].Value = input.Model.Value()
					}
					if !hasErrors {
						m.err = nil
						m.exitCode = nil
						m.isExitting = true
						return m, tea.Quit
					} else {
						m.focused = firstErrorIndex
						m.err = fmt.Errorf("validation errors")
					}
				}
			}
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyEsc:
			m.isExitting = true
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP, tea.KeyUp, tea.KeyLeft:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN, tea.KeyDown, tea.KeyRight:
			m.nextInput()
		}
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		if m.focused <= len(m.inputs)-1 {
			m.inputs[m.focused].Focus()
		}

	// We handle errors just like any other message
	case error:
		fmt.Printf("error: %s\n", msg)
		m.err = msg
		return m, nil
	}

	for i := range m.inputs {
		m.inputs[i].Model, cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

// nextInput focuses the next input field
func (m *FormModel) nextInput() {
	m.focused++
	if m.focused > (len(m.inputs) + len(m.buttons) - 1) {
		m.focused = 0
	}
}

// prevInput focuses the previous input field
func (m *FormModel) prevInput() {
	m.focused--
	// Wrap around
	if m.focused < 0 {
		m.focused = (len(m.inputs) + len(m.buttons) - 1)
	}
}

func (m FormModel) View() string {
	var message bytes.Buffer
	fmt.Fprintf(&message, "📝 %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.title))
	if m.description != "" {
		fmt.Fprintf(&message, "%s\n\n", m.description)
	}
	for i, input := range m.inputs {
		if i == m.focused {
			input.Focus()
		}
		inputDisplay := lipgloss.NewStyle().Render(input.View())
		if m.focused == i {
			inputDisplay = lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiBlue)).Render(inputDisplay)
		}
		fmt.Fprintf(
			&message,
			"%s: %s\n%s\n",
			input.Id,
			inputDisplay,
			lipgloss.NewStyle().Faint(true).Render(input.Description),
		)

		if input.Value != nil {
			if input.Validate != nil {
				input.Err = input.Validate(input.Value.(string))
				if input.Err != nil {
					fmt.Fprintf(&message, "%s", lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiRed)).Render(fmt.Sprintf("❗️ %s\n", input.Err)))
				}
			}
		}
		fmt.Fprintf(&message, "\n")
	}
	if !m.isExitting {
		if m.err != nil {
			fmt.Fprintf(&message, "💡 %s\n\n", lipgloss.NewStyle().Foreground(lipgloss.Color(AnsiYellow)).Render("There are issues with the fields indicated above"))
		}
		buttons := []string{}
		for i, button := range m.buttons {
			buttons = append(buttons, button.Render(m.focused-len(m.inputs) == i))
		}
		fmt.Fprintf(&message, "%s\n", strings.Join(buttons, "\t"))
	}
	return message.String()
}
