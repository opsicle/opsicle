package cli

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type FormOpts struct {
	// Description is an optional description to add to the form
	// which will be displayed below the title
	Description string

	// Fields is a slice of `FormField` instances that will be
	// displayed as fields for a user to input data into
	Fields FormFields

	// InitialFocus is an optional configuration that when set,
	// sets the first focused item
	InitialFocus int

	// Title is the title of the form to be displayed at the top
	// of the form
	Title string
}

func CreateForm(opts FormOpts) *FormModel {

	// initialise the form model

	initWarnings := []error{}
	formModel := &FormModel{
		description: opts.Description,
		title:       opts.Title,
		buttons:     getFormButtons(),
	}

	// process the form fields

	var inputs FormFields
	fieldIds := map[string]struct{}{}
	for _, field := range opts.Fields {
		if _, ok := fieldIds[field.Id]; ok {
			initWarnings = append(initWarnings, fmt.Errorf("id '%s' is duplicated", field.Id))
		}
		inputModel := textinput.New()
		inputModel.Width = 64
		if field.DefaultValue != nil {
			inputModel.SetValue(fmt.Sprintf("%v", field.DefaultValue))
		}
		var validator textinput.ValidateFunc = nil
		switch field.Type {
		case FormFieldBoolean:
			validator = getBooleanValidator(formValidatorOpts{IsRequired: true})
		case FormFieldInteger:
			validator = getIntegerValidator(formValidatorOpts{IsRequired: true})
		case FormFieldFloat:
			validator = getFloatValidator(formValidatorOpts{IsRequired: true})
		case FormFieldEmail:
			validator = getEmailValidator(formValidatorOpts{IsRequired: true})
		case FormFieldSecret:
			inputModel.EchoMode = textinput.EchoPassword
			inputModel.EchoCharacter = '*'
			fallthrough
		case FormFieldString:
			fallthrough
		default:
			validator = getStringValidator(formValidatorOpts{IsRequired: true})
		}
		inputModel.Validate = validator
		inputModel.PlaceholderStyle = formStylePlaceholder
		inputs = append(inputs, FormField{
			DefaultValue: field.DefaultValue,
			Description:  field.Description,
			Id:           field.Id,
			IsRequired:   field.IsRequired,
			Model:        inputModel,
			Label:        field.Label,
			Type:         field.Type,
		})
	}
	formModel.inputs = inputs

	// set the initial focused item

	initialFocus := 0
	if opts.InitialFocus < len(inputs) {
		initialFocus = opts.InitialFocus
	} else if opts.InitialFocus >= len(inputs) {
		initWarnings = append(initWarnings, fmt.Errorf("initial focus of %v is greater than the number of fields", opts.InitialFocus))
	}
	inputs[initialFocus].Focus()
	formModel.focused = initialFocus

	// insert any warnings

	if len(initWarnings) > 0 {
		formModel.initWarnings = errors.Join(initWarnings...)
	}

	return formModel
}

type FormModel struct {
	buttons      []FormButton
	description  string
	err          error
	errTriggered time.Time
	initWarnings error
	exitCode     error
	focused      int
	inputs       []FormField
	isExitting   bool
	title        string
}

// GetValueMap returns a map of the field ID to it's value
func (m *FormModel) GetValueMap() map[string]any {
	output := map[string]any{}
	for _, input := range m.inputs {
		switch input.Type {
		case FormFieldBoolean:
			val, _ := strconv.ParseBool(input.Value())
			output[input.Id] = val
		case FormFieldInteger:
			val, _ := strconv.ParseInt(input.Value(), 10, 64)
			output[input.Id] = val
		case FormFieldFloat:
			val, _ := strconv.ParseFloat(input.Value(), 64)
			output[input.Id] = val
		default:
			output[input.Id] = input.Value()
		}
	}
	return output
}

func (m *FormModel) GetInitWarnings() error {
	return m.initWarnings
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
				break
			}
			switch m.buttons[m.focused-len(m.inputs)].Type {
			case FormButtonCancel:
				m.isExitting = true
				m.exitCode = ErrorUserCancelled
				return m, tea.Quit
			case FormButtonSubmit:
				hasErrors := false
				firstErrorIndex := len(m.inputs)
				for i, input := range m.inputs {
					m.inputs[i].isTouched = true
					if err := input.Validate(input.Model.Value()); err != nil {
						if i < firstErrorIndex {
							firstErrorIndex = i
						}
						m.inputs[i].Err = err
						hasErrors = true
					}
				}
				if !hasErrors {
					m.err = nil
					m.exitCode = nil
					m.isExitting = true
					return m, tea.Quit
				} else {
					m.focused = firstErrorIndex
					m.err = fmt.Errorf("validation errors")
					m.errTriggered = time.Now()
				}
			}
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyEsc:
			m.isExitting = true
			return m, tea.Quit
		case tea.KeyShiftTab, tea.KeyCtrlP, tea.KeyUp, tea.KeyLeft:
			m.prevInput()
		case tea.KeyTab, tea.KeyCtrlN, tea.KeyDown, tea.KeyRight:
			m.nextInput()
		default:
			m.inputs[m.focused].Touch()
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
		m.errTriggered = time.Now()
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
	fmt.Fprintf(&message, "📝 %s\n\n", formStyleTitle.Render(m.title))
	if m.description != "" {
		fmt.Fprintf(&message, "%s\n\n", m.description)
	}
	for i, input := range m.inputs {
		if i == m.focused {
			input.Focus()
		}
		inputDisplay := formStyleInput.Render(input.View())
		if m.focused == i {
			inputDisplay = formStyleInputFocused.Render(inputDisplay)
		}
		isRequired := ""
		if input.IsRequired {
			isRequired = " (*)"
		}
		fmt.Fprintf(
			&message,
			"%s%s: %s\n%s\n",
			formStyleInputLabel.Render(fmt.Sprintf("%s [%s]", input.Label, input.Id)),
			isRequired,
			inputDisplay,
			formStyleFaded.Render(input.Description),
		)

		if input.Validate != nil {
			input.Err = input.Validate(input.Model.Value())
		}
		if input.isTouched && input.Err != nil {
			fmt.Fprintf(&message, "%s", formStyleInputError.Render(fmt.Sprintf("❗️ %s\n", input.Err)))
		}
		fmt.Fprintf(&message, "\n")
	}
	if !m.isExitting {
		if m.err != nil {
			fmt.Fprintf(&message, "💡 %s\n\n", formStyleError.Render("There are issues with the fields indicated above"))
		}
		buttons := []string{}
		for i, button := range m.buttons {
			buttons = append(buttons, button.View(m.focused-len(m.inputs) == i))
		}
		fmt.Fprintf(&message, "%s\n", strings.Join(buttons, "\t"))
	}
	return message.String()
}
