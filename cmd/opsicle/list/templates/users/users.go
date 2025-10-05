package users

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "controller-url",
		Short:        'u',
		DefaultValue: "http://localhost:54321",
		Usage:        "Defines the url where the controller service is accessible at",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "template-id",
		Short:        't',
		DefaultValue: "",
		Usage:        "ID of the template to remove a user from",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "users",
	Aliases: []string{"accounts", "accs", "user", "u"},
	Short:   "Lists users that have access to a template identified by the --template-id flag",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		controllerUrl := viper.GetString("controller-url")
		methodId := "opsicle/list/template/users"
	enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, err := rootCmd.ExecuteC()
			if err != nil {
				return err
			}
			goto enforceAuth
		}

		client, err := controller.NewClient(controller.NewClientOpts{
			ControllerUrl: controllerUrl,
			BearerAuth: &controller.NewClientBearerAuthOpts{
				Token: sessionToken,
			},
			Id: methodId,
		})
		if err != nil {
			return fmt.Errorf("failed to create controller client: %w", err)
		}

		templateId := viper.GetString("template-id")
		if _, err := uuid.Parse(templateId); err != nil {
			cli.PrintBoxedErrorMessage(
				"We didn't receive a valid template ID",
			)
			return fmt.Errorf("invalid template id")
		}

		listTemplateUsersOutput, err := client.ListTemplateUsersV1(controller.ListTemplateUsersV1Input{
			TemplateId: templateId,
		})
		if err != nil {
			return fmt.Errorf("failed to list template users: %w", err)
		}
		templateUsers := listTemplateUsersOutput.Data.Users
		sort.Slice(templateUsers, func(i, j int) bool {
			return strings.Compare(templateUsers[i].Email, templateUsers[j].Email) < 0
		})

		table := cli.NewTable(cli.NewTableOpts{
			Headers: []string{
				"user",
				"delete",
				"execute",
				"invite",
				"update",
				"view",
				"invited by",
				"since",
			},
			IsFullWidth: true,
			Rows: func(t *cli.Table) error {
				for i, user := range templateUsers {
					if err := t.NewRow(
						user.Email,
						user.CanDelete,
						user.CanExecute,
						user.CanInvite,
						user.CanUpdate,
						user.CanView,
						user.CreatedBy,
						user.CreatedAt.Local().Format(cli.TimestampHuman),
					); err != nil {
						return fmt.Errorf("failed to insert row[%v] :%w", i, err)
					}
				}
				return nil
			},
		}).Render()
		fmt.Println(table.GetString())

		return nil
	},
}
