package cli

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type FormFieldType string

const (
	FormFieldBoolean FormFieldType = "boolean"
	FormFieldEmail   FormFieldType = "email"
	FormFieldFloat   FormFieldType = "float"
	FormFieldInteger FormFieldType = "integer"
	FormFieldSecret  FormFieldType = "secret"
	FormFieldString  FormFieldType = "string"
)

type FormFields []FormField

type FormField struct {
	textinput.Model

	// Id is the unique identifier of this field, if there
	// are duplicates, undefined behaviour can be expected
	Id string

	// Label is the label of the field that will be dispalyed
	// to the user
	Label string

	// Description is an optional description to display to
	// the end-user
	Description string

	// DefaultValue is the default value that should be set
	// on the field
	DefaultValue any

	// Type is the intended type of the field
	Type FormFieldType

	// IsLabelHidden indicates whether the label of the field should be
	// hidden from the display
	IsLabelHidden bool

	// IsRequired indicates whether a field is required to be
	// filled up, displays an error to the user when this is
	// set to `true` but the field is empty
	IsRequired bool

	// isTouched is an internally used field to indiciate when
	// a field has been touched
	isTouched bool
}

func (f *FormField) View() string {
	switch f.Type {
	case FormFieldBoolean:
		value, _ := strconv.ParseBool(f.Model.Value())
		display := "[   ]"
		if value {
			display = "[ ✔ ]"
		}
		displayStyle := styleInput
		if f.Model.Focused() {
			displayStyle = styleInputFocused
		}
		return displayStyle.Render(display)
	default:
		return f.Model.View()
	}
}

func (f *FormField) Update(msg tea.Msg) (textinput.Model, tea.Cmd) {
	if !f.Focused() {
		return f.Model.Update(nil)
	}
	switch f.Type {
	case FormFieldBoolean:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeySpace:
				val, _ := strconv.ParseBool(f.Model.Value())
				newVal := strconv.FormatBool(!val)
				f.Model.SetValue(newVal)
				return f.Model.Update(nil)
			}
		}
	}
	return f.Model.Update(msg)
}

func (f *FormField) Touch() {
	f.isTouched = true
}

func (f *FormField) getDescription(maxWidth int) string {
	description := fmt.Sprintf("(type: %v)", f.Type)
	if f.IsRequired {
		description = fmt.Sprintf("%s [REQUIRED]", description)
	}
	description += " " + f.Description
	return wrapString(description, maxWidth)
}

func (f *FormField) getError(maxWidth int) string {
	return styleInputError.Render(wrapString(fmt.Sprintf("❗️ %s", f.Err), maxWidth))
}

func (f *FormField) getDisplay(isFocused bool, maxWidth int) string {
	var message bytes.Buffer
	fieldName := fmt.Sprintf("%s [%s]", f.Label, f.Id)
	inputDisplay := styleInput.Render(f.View())
	if isFocused {
		inputDisplay = styleInputFocused.Render(inputDisplay)
	}
	description := f.getDescription(maxWidth)
	fmt.Fprintf(
		&message,
		"%s: %s\n%s\n",
		styleInputLabel.Render(fieldName),
		wrapString(inputDisplay, maxWidth),
		styleFaded.Render(description),
	)

	if f.isTouched && f.Err != nil {
		fmt.Fprintf(&message, "%s\n", f.getError(maxWidth))
	}
	return message.String()
}
