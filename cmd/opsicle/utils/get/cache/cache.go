package cache

import (
	"fmt"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/config"
	"opsicle/internal/integrations/redis"
	"opsicle/internal/persistence"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{}.Append(config.GetRedisFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Flags: flags,
	Use:   "cache [key]",
	Short: "Retrieves the cache key as specified",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("failed to receive a valid key")
		}
		cacheKey := args[0]
		logrus.Infof("retrieving cache key[%s]...", cacheKey)

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

		cacheVal, err := cache.Get().Get(cacheKey)
		if err != nil {
			if redis.IsNilResult(err) {
				logrus.Errorf("cache key[%s] was not set", cacheKey)
				return nil
			}
			return fmt.Errorf("failed to get cache key[%s]: %w", cacheKey, err)
		}

		fmt.Println(cacheVal)
		return nil
	},
})
