package approver

import (
	"fmt"
	"opsicle/internal/approver"
	"opsicle/internal/cache"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	flags.AddToCommand(Command)
}

var flags cli.Flags = cli.Flags{
	{
		Name:         "basic-auth-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, basic auth is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "basic-auth-username",
		DefaultValue: "",
		Usage:        "the username segment when authenticating with basic auth; when specified, requires '--basic-auth-password' to be set as well",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "basic-auth-password",
		DefaultValue: "",
		Usage:        "the password segment when authenticating with basic auth; when specified, requires '--basic-auth-username' to be set as well",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "bearer-auth-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, bearer auth is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "bearer-auth-token",
		DefaultValue: "",
		Usage:        "the required token when a consumer authenticates with bearer auth",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "ip-allowlist-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, ip allowlist is enabled and blocks all traffic from ip addresses not in the provided list; cidrs are allowed",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "ip-allowlist",
		DefaultValue: []string{},
		Usage:        "specifies remote ip addresses that are allowed to communicate with the server",
		Type:         cli.FlagTypeStringSlice,
	},
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:12345",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "redis-enabled",
		DefaultValue: true,
		Usage:        "when this flag is specified, redis is used as the cache",
		Type:         cli.FlagTypeBool,
	},
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
	{
		Name:         "slack-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, the slack bot is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "slack-app-token",
		DefaultValue: "",
		Usage:        "the slack app token to be used when slack is enabled",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "slack-bot-token",
		DefaultValue: "",
		Usage:        "the slack bot token to be used when slack is enabled",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "telegram-enabled",
		DefaultValue: false,
		Usage:        "when this flag is specified, the telegram bot is enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "telegram-bot-token",
		DefaultValue: "",
		Usage:        "the telegram bot token to be used when telegram is enabled",
		Type:         cli.FlagTypeString,
	},
}

var Command = &cobra.Command{
	Use:     "approver",
	Aliases: []string{"a"},
	Short:   "Starts the approver component",
	Long:    "Starts the approver component which serves as a background job that communicates with the configured component",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		isRedisEnabled := viper.GetBool("redis-enabled")
		logrus.Debugf("redis-enabled status: %v", isRedisEnabled)
		if isRedisEnabled {
			if err := cache.InitRedis(cache.InitRedisOpts{
				Addr:        viper.GetString("redis-addr"),
				Username:    viper.GetString("redis-username"),
				Password:    viper.GetString("redis-password"),
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise redis cache: %w", err)
			}
			logrus.Infof("redis client initialised")
		}

		isSlackEnabled := viper.GetBool("slack-enabled")
		logrus.Debugf("slack-enabled status: %v", isSlackEnabled)
		if isSlackEnabled {
			slackBotToken := viper.GetString("slack-bot-token")
			if slackBotToken == "" {
				return fmt.Errorf("failed to receive a slack bot token")
			}
			slackAppToken := viper.GetString("slack-app-token")
			if slackAppToken == "" {
				return fmt.Errorf("failed to receive a slack app token")
			}
			approver.InitSlackNotifier(approver.InitSlackNotifierOpts{
				AppToken:    slackAppToken,
				BotToken:    slackBotToken,
				ServiceLogs: serviceLogs,
			})
			logrus.Infof("slack notifier initialised")
		}

		isTelegramEnabled := viper.GetBool("telegram-enabled")
		logrus.Debugf("telegram-enabled status: %v", isTelegramEnabled)
		if isTelegramEnabled {
			telegramBotToken := viper.GetString("telegram-bot-token")
			if telegramBotToken == "" {
				return fmt.Errorf("failed to receive a telegram bot token")
			}
			if err := approver.InitTelegramNotifier(approver.InitTelegramNotifierOpts{
				BotToken:    telegramBotToken,
				ServiceLogs: serviceLogs,
			}); err != nil {
				return fmt.Errorf("failed to initialise telegram client: %w", err)
			}
			logrus.Infof("telegram notifier initialised")
		}

		logrus.Debugf("verifying notifiers...")
		if approver.Notifiers == nil {
			return fmt.Errorf("failed to identify a notifier")
		}
		logrus.Debugf("starting notifiers...")
		go approver.Notifiers.StartListening()

		listenAddress := viper.GetString("listen-addr")
		logrus.Debugf("starting http server on addr[%s]...", listenAddress)
		httpServerDone := make(chan common.Done)
		startServerOpts := approver.StartHttpServerOpts{
			Addr:        listenAddress,
			Done:        httpServerDone,
			ServiceLogs: serviceLogs,
		}

		isBasicAuthEnabled := viper.GetBool("basic-auth-enabled")
		if isBasicAuthEnabled {
			logrus.Infof("basic auth is enabled, include credentials in all your requests")
			startServerOpts.BasicAuth = &approver.StartHttpServerBasicAuthOpts{
				Username: viper.GetString("basic-auth-username"),
				Password: viper.GetString("basic-auth-password"),
			}
		}
		isBearerAuthEnabled := viper.GetBool("bearer-auth-enabled")
		if isBearerAuthEnabled {
			logrus.Infof("bearer authentication is enabled, include the 'Authorization: Bearer xyz' header in all your requests")
			startServerOpts.BearerAuth = &approver.StartHttpServerBearerAuthOpts{
				Token: viper.GetString("bearer-auth-token"),
			}
		}

		isIpAllowlistEnabled := viper.GetBool("ip-allowlist-enabled")
		if isIpAllowlistEnabled {
			ipAllowList := viper.GetStringSlice("ip-allowlist")
			logrus.Infof("ip allowlist enabled for cidrs['%s']", strings.Join(ipAllowList, "', '"))
			startServerOpts.IpAllowlist = &approver.StartHttpServerIpAllowlistOpts{
				AllowedIps: ipAllowList,
			}
		}

		approver.StartHttpServer(startServerOpts)

		return cmd.Help()
	},
}
