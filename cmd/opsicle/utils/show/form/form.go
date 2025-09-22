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
			Fields: cli.FormFields{
				{
					Id:           "a",
					Label:        "A",
					Description:  "description a",
					DefaultValue: "a",
				},
				{
					Id:           "b",
					Label:        "B",
					Description:  "this field should be the url to the service you want to call",
					DefaultValue: "b",
					Type:         cli.FormFieldInteger,
				},
				{
					Id:           "c",
					Label:        "C",
					Description:  "description c",
					DefaultValue: "",
					IsRequired:   true,
				},
				{
					Id:           "d",
					Label:        "D",
					Description:  "description d",
					DefaultValue: "",
				},
				{
					Id:           "e",
					Label:        "E",
					Description:  "description e",
					DefaultValue: "e",
				},
			},
			Title: "hey",
		})
		p := tea.NewProgram(form)
		if _, err := p.Run(); err != nil {
			panic(err)
		}
		o, _ := json.MarshalIndent(form.GetData(), "", "  ")
		fmt.Println(string(o))
		return nil
	},
}
