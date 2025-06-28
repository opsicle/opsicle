package automationtemplate

import (
	"fmt"
	"opsicle/internal/automationtemplates"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
}

var Command = &cobra.Command{
	Use:     "automationtemplate <path-to-template-file>",
	Aliases: []string{"template"},
	Short:   "Validates an AutomationTemplate resource",
	RunE: func(cmd *cobra.Command, args []string) error {
		resourceIsSpecified := false
		resourcePath := ""
		if len(args) > 0 {
			resourcePath = args[0]
			resourceIsSpecified = true
		}
		if !resourceIsSpecified {
			return fmt.Errorf("failed to receive a <path-to-template-file")
		}
		fi, err := os.Stat(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to check for existence of file at path[%s]: %s", resourcePath, err)
		}
		if fi.IsDir() {
			return fmt.Errorf("failed to get a file at path[%s]: got a directory", resourcePath)
		}
		automationTemplate, err := automationtemplates.LoadFromFile(resourcePath)
		if err != nil {
			return fmt.Errorf("failed to load automation template from path[%s]: %s", resourcePath, err)
		}
		o, _ := yaml.Marshal(automationTemplate)
		fmt.Println(string(o))
		return nil
	},
}
