package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
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

	MaxHeight int
	MaxWidth  int
}

func CreateForm(opts FormOpts) *FormModel {

	// initialise the form model

	initWarnings := []error{}
	formModel := &FormModel{
		description:    opts.Description,
		inputLineIndex: []int{},
		title:          opts.Title,
		buttons:        getFormButtons(),
	}

	formModel.terminalDimensions.width, formModel.terminalDimensions.height, _ = term.GetSize(int(os.Stdout.Fd()))
	formModel.dimensions.height = formModel.terminalDimensions.height
	formModel.dimensions.width = formModel.terminalDimensions.width
	if opts.MaxHeight != 0 && formModel.terminalDimensions.height > opts.MaxHeight {
		formModel.dimensions.height = opts.MaxHeight
	}
	if opts.MaxWidth != 0 && formModel.terminalDimensions.width > opts.MaxWidth {
		formModel.dimensions.width = opts.MaxWidth
	}

	// process the form fields

	var inputs FormFields
	fieldIds := map[string]struct{}{}
	currentLineIndex := 0
	formModel.inputsRaw = opts.Fields
	for _, field := range formModel.inputsRaw {
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
		description := wrapString(field.Description, formModel.dimensions.width)
		formModel.inputLineIndex = append(formModel.inputLineIndex, currentLineIndex)
		currentLineIndex += countLines(description) + 1
		inputs = append(inputs, FormField{
			DefaultValue: field.DefaultValue,
			Description:  description,
			Id:           field.Id,
			IsRequired:   field.IsRequired,
			Model:        inputModel,
			Label:        field.Label,
			Type:         field.Type,
		})
	}
	formModel.inputs = inputs

	formModel.getViewContent()
	contentHeight := formModel.lastKnownHeight
	if contentHeight > formModel.dimensions.height {
		headerHeight := countLines(fmt.Sprintf("%s\n%s", formModel.getTitle(), formModel.getDescription()))
		viewportInstance := viewport.New(formModel.dimensions.width, formModel.dimensions.height-headerHeight)
		formModel.viewport = &viewportInstance
	}

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

type FormModelDimensions struct {
	height int
	width  int
}

type FormModel struct {
	buttons     []FormButton
	description string

	// dimensions is the most updated dimensions for this model
	dimensions     FormModelDimensions
	err            error
	errTriggered   time.Time
	initWarnings   error
	exitCode       error
	focused        int
	inputLineIndex []int
	inputs         []FormField
	inputsRaw      []FormField

	// isExitting is internally used to indicate that user has
	// either submitted or cancelled the form
	isExitting bool

	lastKnownHeight int

	// terminalDimensions contains the most updated dimensions
	// for the terminal this model is running in
	terminalDimensions FormModelDimensions

	// userDimensions contains the dimensions intended by the
	// user, if the `terminalDimensions` are more constrained
	// than these dimensions, the `terminalDimensions` are used,
	// otherwise, these will be used
	userDimensions FormModelDimensions

	// title is a bolded text displayed at the top of the form
	title string

	// viewport is a viewport that becomes non-nil when the
	// height of the form exceeds that of
	viewport *viewport.Model
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
	viewportCount := 0
	if m.viewport != nil {
		viewportCount++
	}
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs)+viewportCount)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalDimensions.width, m.terminalDimensions.height, _ = term.GetSize(int(os.Stdout.Fd()))
		currentLineIndex := 0
		for i, input := range m.inputsRaw {
			m.inputs[i].Description = wrapString(input.Description, m.terminalDimensions.width)
			m.inputLineIndex[i] = currentLineIndex
			currentLineIndex += countLines(m.inputs[i].Description) + 1
		}
		if m.viewport != nil {
			m.viewport.Width = m.terminalDimensions.width
		}
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
		case tea.KeyLeft:
			if m.focused < len(m.inputs) {
				break
			}
			fallthrough
		case tea.KeyShiftTab, tea.KeyCtrlP, tea.KeyUp:
			m.prevInput()
		case tea.KeyRight:
			if m.focused < len(m.inputs) {
				break
			}
			fallthrough
		case tea.KeyTab, tea.KeyCtrlN, tea.KeyDown:
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

	if m.viewport != nil {
		m.viewport.SetContent(m.getViewContent())
		if m.focused == 0 {
			m.viewport.GotoTop()
		} else if m.focused < len(m.inputs) {
			m.viewport.SetYOffset(m.inputLineIndex[m.focused])
		} else {
			m.viewport.GotoBottom()
		}
		// issue is here, this update here is processing the down/up but not left/right
		updatedViewport, cmd := m.viewport.Update(nil)
		m.viewport = &updatedViewport
		cmds = append(cmds, cmd)
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
	if m.focused < 0 {
		m.focused = (len(m.inputs) + len(m.buttons) - 1)
	}
}

func (m *FormModel) getDescription() string {
	if m.description != "" {
		return fmt.Sprintf("%s\n", wrapString(m.description, m.dimensions.width))
	}
	return ""
}

func (m *FormModel) getHeader() string {
	title := m.getTitle()
	description := m.getDescription()
	if description != "" {
		title += "\n"
	}
	return title + description
}

func (m *FormModel) getTitle() string {
	yOffset := 0
	supposedYOffset := 0
	if m.viewport != nil {
		yOffset = m.viewport.YOffset
		if m.focused < len(m.inputs) {
			supposedYOffset = m.inputLineIndex[m.focused]
		}
	}
	return fmt.Sprintf("📝 %v:%v:%v %s\n", m.focused, yOffset, supposedYOffset, wrapString(formStyleTitle.Render(m.title), m.dimensions.width))
}

func (m *FormModel) getViewContent() string {
	var message bytes.Buffer
	if m.viewport == nil {
		fmt.Fprint(&message, m.getHeader()+"\n")
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
			fmt.Fprintf(&message, "%s\n", formStyleInputError.Render(wrapString(fmt.Sprintf("❗️ %s", input.Err), m.dimensions.width)))
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
	m.lastKnownHeight = countLines(message.String())
	return message.String()
}

func (m FormModel) View() string {
	if m.viewport != nil {
		var header bytes.Buffer
		fmt.Fprint(&header, m.getHeader())
		return fmt.Sprintf("%s\n%s", header.String(), m.viewport.View())
	}
	return m.getViewContent()
}
