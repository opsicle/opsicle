package cli

import "fmt"

const (
	FormButtonCancel FormButtonType = "cancel"
	FormButtonSubmit FormButtonType = "submit"
)

func getFormButtons() FormButtons {
	return FormButtons{
		CreateFormButton("Submit", FormButtonSubmit),
		CreateFormButton("Cancel", FormButtonCancel),
	}
}

func CreateFormButton(label string, t FormButtonType) FormButton {
	return FormButton{
		Label: label,
		Type:  t,
	}
}

type FormButtons []FormButton

type FormButton struct {
	Label string
	Type  FormButtonType
}

type FormButtonType string

func (fb *FormButton) View(isSelected bool) string {
	displayStyle := styleInput
	if isSelected {
		displayStyle = styleInputFocused
	}
	return displayStyle.Render(fmt.Sprintf("[ %s ]", fb.Label))
}
