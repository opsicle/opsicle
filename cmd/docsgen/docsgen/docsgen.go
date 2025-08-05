package docsgen

import (
	"fmt"
	"opsicle/cmd/opsicle"
	"opsicle/internal/cli"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "docs-path",
		DefaultValue: "./docs/cli",
		Usage:        "defines the path to the documentation",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:   "docsgen",
	Short: "Retrieves documentation in Markdown",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		docsPath := viper.GetString("docs-path")
		logrus.Infof("generating documentation at path[%s]", docsPath)
		err := os.MkdirAll(docsPath, 0755)
		if err != nil {
			return err
		}
		commandMap := map[string]bool{}
		if err := doc.GenMarkdownTreeCustom(opsicle.Command, docsPath, func(in string) string {
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
		for k, _ := range commandMap {
			commandList = append(commandList, k)
		}
		var sidebar strings.Builder
		sort.Strings(commandList)
		sidebar.WriteString("* [Home](/)\n")
		sidebar.WriteString("* [opsicle](cli/opsicle)\n")
		for _, cmd := range commandList {
			commandName := strings.Split(cmd, ".")
			commandParts := strings.Split(commandName[0], "_")
			if len(commandParts) > 1 {
				for i := 0; i < len(commandParts)-1; i++ {
					sidebar.WriteString("  ")
				}
				sidebar.WriteString(fmt.Sprintf("* [%s](cli/%s)\n", commandParts[len(commandParts)-1], cmd))
			}
		}
		os.WriteFile(path.Join(docsPath, "_sidebar.md"), []byte(sidebar.String()), 0755)
		return nil
	},
}
