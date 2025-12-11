package cache

import (
	"fmt"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/persistence"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "ttl-duration",
		Short:        't',
		DefaultValue: 30 * time.Second,
		Usage:        "defines the duration which the set key/value pair should live",
		Type:         cli.FlagTypeDuration,
	},
}.Append(config.GetRedisFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Flags: flags,
	Use:   "cache [key] [value]",
	Short: "Sets the cache key as specified",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("failed to receive a valid key and value")
		}
		cacheKey := args[0]
		cacheValue := args[1]
		logrus.Infof("setting cache key[%s] to value[%s]...", cacheKey, cacheValue)

		serviceLogs := opts.GetServiceLogs()

		redisInstance := persistence.NewRedis(
			persistence.RedisConnectionOpts{
				AppName: "opsicle/utils/get/cache",
				Addr:    viper.GetString(config.RedisAddr),
			},
			persistence.RedisAuthOpts{
				Username: viper.GetString(config.RedisUsername),
				Password: viper.GetString(config.RedisPassword),
			},
			&serviceLogs,
		)

		cache.InitRedis(cache.InitRedisOpts{
			RedisConnection: redisInstance,
			ServiceLogs:     serviceLogs,
		})

		cacheTtl := viper.GetDuration("ttl-duration")
		if err := cache.Get().Set(cacheKey, cacheValue, cacheTtl); err != nil {
			return fmt.Errorf("failed to set cache key[%s]: %w", cacheKey, err)
		}

		return nil
	},
})
