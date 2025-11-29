package cache

import (
	"fmt"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/persistence"

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

var Command = cli.NewCommand(cli.CommandOpts{
	Name:    "check.cache",
	Flags:   flags,
	Use:     "cache",
	Aliases: []string{"c"},
	Short:   "Checks cache connectivity",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		appName := opts.GetFullname()
		serviceLogs := opts.GetServiceLogs()

		logrus.Infof("verifying cache connectivity...")
		redisInstance := persistence.NewRedis(
			persistence.RedisConnectionOpts{
				AppName: appName,
				Addr:    viper.GetString("redis-addr"),
			},
			persistence.RedisAuthOpts{
				Username: viper.GetString("redis-username"),
				Password: viper.GetString("redis-password"),
			},
			&serviceLogs,
		)
		if err := redisInstance.Init(); err != nil {
			return fmt.Errorf("failed to connect to redis: %w", err)
		}

		if err := cache.InitRedis(cache.InitRedisOpts{
			RedisConnection: redisInstance,
			ServiceLogs:     serviceLogs,
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
})
