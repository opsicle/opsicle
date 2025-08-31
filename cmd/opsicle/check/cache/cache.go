package cache

import (
	"fmt"
	"opsicle/internal/cache"
	"opsicle/internal/cli"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "redis-addr",
		DefaultValue: "localhost:6379",
		Usage:        "defines the hostname (including port) of the redis server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-username",
		DefaultValue: "opsicle",
		Usage:        "defines the username used to login to redis",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-password",
		DefaultValue: "password",
		Usage:        "defines the password used to login to redis",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "cache",
	Aliases: []string{"c"},
	Short:   "Checks cache connectivity",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("verifying cache connectivity...")
		if err := cache.InitRedis(cache.InitRedisOpts{
			Addr:     viper.GetString("redis-addr"),
			Username: viper.GetString("redis-username"),
			Password: viper.GetString("redis-password"),
		}); err != nil {
			return fmt.Errorf("failed to initialise cache: %w", err)
		}
		if err := cache.Get().Ping(); err != nil {
			return fmt.Errorf("failed to establish connection to cache: %w", err)
		}
		defer cache.Get().Close()
		cli.PrintBoxedSuccessMessage(fmt.Sprintf(
			"Successfully connected to cache at address[%s]",
			viper.GetString("redis-addr"),
		))
		return nil
	},
}
