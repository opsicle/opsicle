package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

// Flags defines a collection of flags as a slice
type Flags []FlagData

// AddToCommand is a convenience function to run the `.AddToCommand`
// method on all children in this slice
func (f Flags) AddToCommand(command *cobra.Command, persistent ...bool) {
	for _, g := range f {
		g.AddToCommand(command, persistent...)
	}
}

func (f Flags) Append(more Flags) Flags {
	f = append(f, more...)
	return f
}

// BindViper is a convenience function to run the `.BindViper`
// method on all children in this slice
func (f Flags) BindViper(command *cobra.Command, persistent ...bool) {
	for _, g := range f {
		g.BindViper(command, persistent...)
	}
}

// FlagData represents a logical flag; when being processed,
// the `.Name` property will be used as the `viper` reference
// and will be normalised to `kebab-case“
type FlagData struct {
	Name         string
	Short        rune
	DefaultValue any
	Usage        string
	Type         FlagType
}

// FlagType provides a restriction to the type of flag and
// nudges users to use a defined FlagType from this package
type FlagType string

// AddToCommand adds a flag to the provided `command` instance,
// this should be done during the `init()` method. Panics if the
// `.Type` property is not something we recognise
func (f *FlagData) AddToCommand(command *cobra.Command, persistent ...bool) {
	var flags *pflag.FlagSet
	if len(persistent) > 0 && persistent[0] {
		flags = command.PersistentFlags()
	} else {
		flags = command.Flags()
	}
	switch f.Type {

	case FlagTypeBool:
		if f.Short != 0 {
			flags.BoolP(f.Name, string(f.Short), f.DefaultValue.(bool), f.Usage)
			break
		}
		flags.Bool(f.Name, f.DefaultValue.(bool), f.Usage)

	case FlagTypeDuration:
		if f.Short != 0 {
			flags.DurationP(f.Name, string(f.Short), f.DefaultValue.(time.Duration), f.Usage)
			break
		}
		flags.Duration(f.Name, f.DefaultValue.(time.Duration), f.Usage)

	case FlagTypeFloat:
		if f.Short != 0 {
			flags.Float64P(f.Name, string(f.Short), f.DefaultValue.(float64), f.Usage)
			break
		}
		flags.Float64(f.Name, f.DefaultValue.(float64), f.Usage)

	case FlagTypeInteger:
		if f.Short != 0 {
			flags.IntP(f.Name, string(f.Short), f.DefaultValue.(int), f.Usage)
			break
		}
		flags.Int(f.Name, f.DefaultValue.(int), f.Usage)

	case FlagTypeString:
		if f.Short != 0 {
			flags.StringP(f.Name, string(f.Short), f.DefaultValue.(string), f.Usage)
			break
		}
		flags.String(f.Name, f.DefaultValue.(string), f.Usage)

	case FlagTypeStringSlice:
		if f.Short != 0 {
			flags.StringSliceP(f.Name, string(f.Short), f.DefaultValue.([]string), f.Usage)
			break
		}
		flags.StringSlice(f.Name, f.DefaultValue.([]string), f.Usage)
	default:
		panic(fmt.Sprintf("unknown FlagType[%s]", f.Type))
	}
}

// BindViper binds the current flag to viper assuming the
// .Name property as the name being applied to the `pflag.FlagSet`
// property in the `command` argument. This should be done
// during the `cobra.Command.PreRun` phase to avoid overwriting
// variables defined in other commands
func (f *FlagData) BindViper(command *cobra.Command, persistent ...bool) {
	var flags *pflag.FlagSet
	if len(persistent) > 0 && persistent[0] {
		flags = command.PersistentFlags()
	} else {
		flags = command.Flags()
	}
	viper.BindPFlag(f.Name, flags.Lookup(f.Name))
	viper.BindEnv(f.Name)
}
