package cli

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Flags []FlagData

func (f Flags) AddToCommand(command *cobra.Command) {
	for _, g := range f {
		g.AddToCommand(command)
	}
}

func (f Flags) BindViper(command *cobra.Command) {
	for _, g := range f {
		g.BindViper(command)
	}
}

type FlagData struct {
	Name         string
	Short        rune
	DefaultValue any
	Usage        string
	Type         FlagType
}

type FlagType string

func (f *FlagData) AddToCommand(command *cobra.Command) {
	switch c.Type {
	case FlagTypeBool:
		if c.Short != 0 {
			command.Flags().BoolP(c.Name, string(c.Short), c.DefaultValue.(bool), c.Usage)
			break
		}
		command.Flags().Bool(c.Name, c.DefaultValue.(bool), c.Usage)
	case FlagTypeDuration:
		if c.Short != 0 {
			command.Flags().DurationP(c.Name, string(c.Short), c.DefaultValue.(time.Duration), c.Usage)
			break
		}
		command.Flags().Duration(c.Name, c.DefaultValue.(time.Duration), c.Usage)
	case FlagTypeFloat:
		if c.Short != 0 {
			command.Flags().Float64P(c.Name, string(c.Short), c.DefaultValue.(float64), c.Usage)
			break
		}
		command.Flags().Float64(c.Name, c.DefaultValue.(float64), c.Usage)
	case FlagTypeInteger:
		if c.Short != 0 {
			command.Flags().IntP(c.Name, string(c.Short), c.DefaultValue.(int), c.Usage)
			break
		}
		command.Flags().Int(c.Name, c.DefaultValue.(int), c.Usage)
	case FlagTypeString:
		if c.Short != 0 {
			command.Flags().StringP(c.Name, string(c.Short), c.DefaultValue.(string), c.Usage)
			break
		}
		command.Flags().String(c.Name, c.DefaultValue.(string), c.Usage)
	case FlagTypeStringSlice:
		if c.Short != 0 {
			command.Flags().StringSliceP(c.Name, string(c.Short), c.DefaultValue.([]string), c.Usage)
			break
		}
		command.Flags().StringSlice(c.Name, c.DefaultValue.([]string), c.Usage)
	}
}

func (f *FlagData) BindViper(command *cobra.Command) {
	viper.BindPFlag(f.Name, command.Flags().Lookup(f.Name))
	viper.BindEnv(f.Name)
}
