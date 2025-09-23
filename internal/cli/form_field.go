package cli

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type FormFieldType string

const (
	FormFieldBoolean = "boolean"
	FormFieldEmail   = "email"
	FormFieldFloat   = "float"
	FormFieldInteger = "integer"
	FormFieldSecret  = "secret"
	FormFieldString  = "string"
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
		displayStyle := formStyleInput
		if f.Model.Focused() {
			displayStyle = formStyleInputFocused
		}
		return displayStyle.Render(display)
	default:
		return f.Model.View()
	}
}

func (f *FormField) Update(msg tea.Msg) (textinput.Model, tea.Cmd) {
	if !f.Focused() {
		return f.Model, nil
	}
	switch f.Type {
	case FormFieldBoolean:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeySpace:
				f.Touch()
				val, _ := strconv.ParseBool(f.Model.Value())
				newVal := strconv.FormatBool(!val)
				f.Model.SetValue(newVal)
				return f.Model, nil
			}
		}
	}
	return f.Model.Update(msg)
}

func (f *FormField) Touch() {
	f.isTouched = true
}
