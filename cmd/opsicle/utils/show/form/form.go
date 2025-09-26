package form

import (
	"encoding/json"
	"fmt"
	"opsicle/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
}

var Command = &cobra.Command{
	Use:     "form",
	Aliases: []string{"f"},
	Short:   "Shows a form UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("show a form")
		form := cli.CreateForm(cli.FormOpts{
			Description: "This is a sample form",
			Fields: cli.FormFields{
				{
					Id:           "string_with_default_value",
					Label:        "A",
					Description:  "this is a string field with a default value",
					DefaultValue: "default",
					Type:         cli.FormFieldString,
				},
				{
					Id:          "string_without_default_value",
					Label:       "B",
					Description: "this is a string field without a default value and a really really really really really really really really really long description to verify that line breaks don't break the cursor focus",
					Type:        cli.FormFieldString,
					IsRequired:  true,
				},
				{
					Id:           "int_with_default_value",
					Label:        "C",
					Description:  "this is a int field with a default value",
					DefaultValue: 42,
					Type:         cli.FormFieldInteger,
				},
				{
					Id:          "int_without_default_value",
					Label:       "D",
					Description: "this is a int field without a default value",
					Type:        cli.FormFieldInteger,
					IsRequired:  true,
				},
				{
					Id:           "float_with_default_value",
					Label:        "E",
					Description:  "this is a float field with a default value",
					DefaultValue: 3.142,
					Type:         cli.FormFieldFloat,
				},
				{
					Id:          "float_without_default_value",
					Label:       "F",
					Description: "this is a float field without a default value",
					Type:        cli.FormFieldFloat,
					IsRequired:  true,
				},
				{
					Id:           "bool_with_default_value",
					Label:        "G",
					Description:  "this is a bool field with a default value",
					DefaultValue: 3.142,
					Type:         cli.FormFieldBoolean,
				},
				{
					Id:          "bool_without_default_value",
					Label:       "H",
					Description: "this is a bool field without a default value",
					Type:        cli.FormFieldBoolean,
					IsRequired:  true,
				},
				{
					Id:           "email_with_default_value",
					Label:        "I",
					Description:  "this is a email field with a default value",
					DefaultValue: "someone@somewhere.com",
					Type:         cli.FormFieldEmail,
				},
				{
					Id:          "email_without_default_value",
					Label:       "J",
					Description: "this is a email field without a default value",
					Type:        cli.FormFieldEmail,
					IsRequired:  true,
				},
				{
					Id:           "secret_with_default_value",
					Label:        "K",
					Description:  "this is a secret field with a default value",
					DefaultValue: "password",
					Type:         cli.FormFieldSecret,
				},
				{
					Id:          "secret_without_default_value",
					Label:       "L",
					Description: "this is a secret field without a default value",
					Type:        cli.FormFieldSecret,
					IsRequired:  true,
				},
			},
			Title: "hey",
		})
		if err := form.GetInitWarnings(); err != nil {
			return fmt.Errorf("failed to create form as expected: %w", err)
		}
		p := tea.NewProgram(form)
		if _, err := p.Run(); err != nil {
			panic(err)
		}
		o, _ := json.MarshalIndent(form.GetValueMap(), "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
