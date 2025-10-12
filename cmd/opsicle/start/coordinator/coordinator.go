package coordinator

import (
	"fmt"
	"opsicle/internal/audit"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/queue"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "listen-addr",
		DefaultValue: "0.0.0.0:12345",
		Usage:        "specifies the listen address of the server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-addr",
		DefaultValue: "localhost:4222",
		Usage:        "Specifies the hostname (including port) of the NATS server",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-username",
		DefaultValue: "opsicle",
		Usage:        "Specifies the username used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "nats-password",
		DefaultValue: "password",
		Usage:        "Specifies the password used to login to NATS",
		Type:         cli.FlagTypeString,
	},
	{
		Name: "nats-nkey-value",
		// this default value is the development nkey, this value must be aligned
		// to the one in `./docker-compose.yml` in the root of the repository
		DefaultValue: "SUADZTA4VJHBCO7K75DQ3IN7KZGWHKEI26D2IYEABRN5TXXYHXLWNDYT4A",
		Usage:        "Specifies the nkey used to login to NATS",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "coordinator",
	Aliases: []string{"C"},
	Short:   "Starts the coordinator component",
	Long:    "Starts the coordinator component which serves as the API layer that worker interfaces can connect to to receive jobs",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debugf("starting logging engine...")
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		logrus.Debugf("started logging engine")

		hostname, _ := os.Hostname()
		userId := os.Getuid()
		serviceId := fmt.Sprintf("opsicle/coordinator@%v@%s", userId, hostname)

		/*
		   ___  _   _ ___ _   _ ___
		  / _ \| | | | __| | | | __|
		 | (_) | |_| | _|| |_| | _|
		  \__\_\\___/|___|\___/|___|

		*/
		logrus.Infof("establishing connection to queue...")
		nats, err := queue.InitNats(queue.InitNatsOpts{
			Id:          serviceId,
			Addr:        viper.GetString("nats-addr"),
			Username:    viper.GetString("nats-username"),
			Password:    viper.GetString("nats-password"),
			NKey:        viper.GetString("nats-nkey-value"),
			ServiceLogs: serviceLogs,
		})
		if err != nil {
			return fmt.Errorf("failed to initialise nats queue: %w", err)
		}
		if err := nats.Connect(); err != nil {
			return fmt.Errorf("failed to connect to nats: %w", err)
		}
		logrus.Debugf("established connection to queue")
		audit.Log(audit.LogEntry{
			EntityId:     fmt.Sprintf("%v@%s", userId, hostname),
			EntityType:   audit.CoordinatorEntity,
			Verb:         audit.Connect,
			ResourceId:   viper.GetString("nats-addr"),
			ResourceType: audit.CacheResource,
		})

		return cmd.Help()
	},
}
