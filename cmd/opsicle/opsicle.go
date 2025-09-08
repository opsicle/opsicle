package opsicle

import (
	"fmt"
	"opsicle/cmd/opsicle/admin"
	"opsicle/cmd/opsicle/check"
	"opsicle/cmd/opsicle/create"
	"opsicle/cmd/opsicle/get"
	"opsicle/cmd/opsicle/invite"
	"opsicle/cmd/opsicle/join"
	"opsicle/cmd/opsicle/leave"
	"opsicle/cmd/opsicle/list"
	"opsicle/cmd/opsicle/login"
	"opsicle/cmd/opsicle/logout"
	"opsicle/cmd/opsicle/register"
	"opsicle/cmd/opsicle/remove"
	"opsicle/cmd/opsicle/reset"
	"opsicle/cmd/opsicle/run"
	"opsicle/cmd/opsicle/start"
	"opsicle/cmd/opsicle/submit"
	"opsicle/cmd/opsicle/update"
	"opsicle/cmd/opsicle/utils"
	"opsicle/cmd/opsicle/validate"
	"opsicle/cmd/opsicle/verify"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/config"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

var availableOutputs = []string{
	"text",
	"json",
}

var availableLogLevels = []string{
	string(common.LogLevelTrace),
	string(common.LogLevelDebug),
	string(common.LogLevelInfo),
	string(common.LogLevelWarn),
	string(common.LogLevelError),
}

var persistentFlags cli.Flags = cli.Flags{
	{
		Name:         "config",
		Short:        'C',
		DefaultValue: "~/.opsicle/config",
		Usage:        "Defines the location of the global configuration used",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "log-level",
		Short:        'l',
		DefaultValue: "info",
		Usage:        fmt.Sprintf("Sets the log level (one of [%s])", strings.Join(availableLogLevels, ", ")),
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "output",
		Short:        'o',
		DefaultValue: "text",
		Usage:        fmt.Sprintf("Sets the output format where applicable (one of [%s])", strings.Join(availableOutputs, ", ")),
		Type:         cli.FlagTypeString,
	},
}

var flags cli.Flags = cli.Flags{
	{
		Name:         "docs",
		DefaultValue: false,
		Usage:        "When this flag is specified, generates Markdown documentation for the CLI application",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "docs-path",
		DefaultValue: "./docs/cli",
		Usage:        "Specifies the location to generate documentation in",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	cobra.AddTemplateFunc("prependText", func() string {
		return cli.Logo + "\n"
	})
	Command.SetHelpTemplate(`{{ prependText }}` + Command.HelpTemplate())
	Command.SetVersionTemplate(cli.Logo + "\n" + `{{with .DisplayName}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}`)

	Command.AddCommand(admin.Command)
	Command.AddCommand(check.Command)
	Command.AddCommand(create.Command)
	Command.AddCommand(get.Command)
	Command.AddCommand(invite.Command)
	Command.AddCommand(join.Command)
	Command.AddCommand(leave.Command)
	Command.AddCommand(list.Command)
	Command.AddCommand(login.Command)
	Command.AddCommand(logout.Command)
	Command.AddCommand(register.Command)
	Command.AddCommand(remove.Command)
	Command.AddCommand(reset.Command)
	Command.AddCommand(run.Command)
	Command.AddCommand(start.Command)
	Command.AddCommand(submit.Command)
	Command.AddCommand(update.Command)
	Command.AddCommand(utils.Command)
	Command.AddCommand(validate.Command)
	Command.AddCommand(verify.Command)
	Command.SilenceErrors = true
	Command.SilenceUsage = true

	persistentFlags.AddToCommand(Command, true)
	flags.AddToCommand(Command)

	logrus.SetOutput(os.Stderr)
	cobra.OnInitialize(func() {
		persistentFlags.BindViper(Command, true)
		flags.BindViper(Command)
		cli.InitLogging(viper.GetString("log-level"))
		configPath := viper.GetString("config")
		logrus.Debugf("using configuration at path[%s]", configPath)
		config.LoadGlobal(configPath)
	})

	cli.InitConfig()
}

var Command = &cobra.Command{
	Use:     "opsicle",
	Short:   "Runbook automations by Platform Engineers for Platform Engineers",
	Version: config.GetVersion(),
	Long:    "Runbook automations by Platform Engineers for Platform Engineers",
	RunE: func(cmd *cobra.Command, args []string) error {
		isGenerateDocs := viper.GetBool("docs")
		if isGenerateDocs {
			docsPath := viper.GetString("docs-path")
			logrus.Infof("generating documentation at path[%s]", docsPath)
			err := os.MkdirAll(docsPath, 0755)
			if err != nil {
				return err
			}
			commandMap := map[string]bool{}
			if err := doc.GenMarkdownTreeCustom(cmd, docsPath, func(in string) string {
				logrus.Infof("filePrepender: %s", in)
				return ""
			}, func(in string) string {
				logrus.Infof("linkHandler: %s", in)
				commandMap[in] = true
				return fmt.Sprintf("cli/%s", in)
			}); err != nil {
				return fmt.Errorf("failed to generate markdown tree")
			}
			commandList := []string{}
			for k := range commandMap {
				commandList = append(commandList, k)
			}
			var sidebar strings.Builder
			sort.Strings(commandList)
			sidebar.WriteString("* [🏘 Home](/)\n")
			sidebar.WriteString("* [opsicle](cli/opsicle \"Opsicle CLI\")\n")
			for _, cmd := range commandList {
				commandName := strings.Split(cmd, ".")
				commandParts := strings.Split(commandName[0], "_")
				if len(commandParts) > 1 {
					for i := 0; i < len(commandParts)-1; i++ {
						sidebar.WriteString("  ")
					}
					sidebar.WriteString(fmt.Sprintf("* [%s](cli/%s \"Opsicle CLI: %s\")\n", commandParts[len(commandParts)-1], cmd, strings.Join(commandParts, " ")))
				}
			}
			os.WriteFile(path.Join(docsPath, "_sidebar.md"), []byte(sidebar.String()), 0755)
			return nil
		}

		return cmd.Help()
	},
}
