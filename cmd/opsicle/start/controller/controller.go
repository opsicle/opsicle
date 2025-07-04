package controller

import (
	"fmt"
	"opsicle/internal/common"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	currentFlag := "fs-storage-path"
	Command.PersistentFlags().StringP(
		currentFlag,
		"p",
		"./.opsicle",
		"specifies the path to a directory where Opsicle data resides",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "db-host"
	Command.PersistentFlags().StringP(
		currentFlag,
		"H",
		"localhost:5432",
		"specifies the hostname (including port) of the database",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "db-name"
	Command.PersistentFlags().StringP(
		currentFlag,
		"N",
		"opsicle",
		"specifies the name of the central database schema",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "db-user"
	Command.PersistentFlags().StringP(
		currentFlag,
		"U",
		"opsicle",
		"specifies the username to use to login",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "db-password"
	Command.PersistentFlags().StringP(
		currentFlag,
		"P",
		"opsicle",
		"specifies the password to use to login",
	)
	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)

	currentFlag = "storage-mode"
	Command.PersistentFlags().StringP(
		currentFlag,
		"s",
		common.StorageFilesystem,
		fmt.Sprintf("specifies what type of storage we are using, one of ['%s']", strings.Join(common.Storages, "'")),
	)

	viper.BindPFlag(currentFlag, Command.PersistentFlags().Lookup(currentFlag))
	viper.BindEnv(currentFlag)
}

var Command = &cobra.Command{
	Use:     "controller",
	Aliases: []string{"c"},
	Short:   "Starts the controller component",
	Long:    "Starts the controller component which serves as the API layer that user interfaces can connect to to perform actions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
