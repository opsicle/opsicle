package user

import (
	"fmt"
	"opsicle/internal/auth"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/controller/models"
	"opsicle/internal/database"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flags cli.Flags = cli.Flags{
	{
		Name:         "email",
		DefaultValue: "test@opsicle.io",
		Usage:        "the email address you are signing up to Opsicle with",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "password",
		DefaultValue: "P@ssw0rd!!",
		Usage:        "the password for your account to be used with your email address to authenticate",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "type",
		DefaultValue: string(models.TypeUser),
		Usage:        fmt.Sprintf("The type of user to create (one of ['%s'])", strings.Join([]string{string(models.TypeUser), string(models.TypeSupportUser), string(models.TypeSystemAdmin)}, "', '")),
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "with-mfa",
		DefaultValue: false,
		Usage:        "Indicates whether MFA should be automatically enabled",
		Type:         cli.FlagTypeBool,
	},
	{
		Name:         "mysql-host",
		Short:        'H',
		DefaultValue: "127.0.0.1",
		Usage:        "specifies the hostname of the database",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-port",
		Short:        'P',
		DefaultValue: "3306",
		Usage:        "specifies the port which the database is listening on",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-database",
		Short:        'N',
		DefaultValue: "opsicle",
		Usage:        "specifies the name of the central database schema",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-user",
		Short:        'U',
		DefaultValue: "opsicle",
		Usage:        "specifies the username to use to login",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "mysql-password",
		Short:        'p',
		DefaultValue: "password",
		Usage:        "specifies the password to use to login",
		Type:         cli.FlagTypeString,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "user",
	Aliases: []string{"user", "u"},
	Short:   "Creates a user directly via the database, bypassing email verification",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debugf("starting logging engine...")
		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)
		logrus.Debugf("started logging engine")

		logrus.Infof("establishing connection to database...")
		databaseConnection, err := database.ConnectMysql(database.ConnectOpts{
			ConnectionId: "opsicle/controller",
			Host:         viper.GetString("mysql-host"),
			Port:         viper.GetInt("mysql-port"),
			Username:     viper.GetString("mysql-user"),
			Password:     viper.GetString("mysql-password"),
			Database:     viper.GetString("mysql-database"),
		})
		if err != nil {
			return fmt.Errorf("failed to establish connection to database: %s", err)
		}
		defer databaseConnection.Close()
		logrus.Debugf("established connection to database")

		email := viper.GetString("email")
		password := viper.GetString("password")

		if err := models.CreateUserV1(models.CreateUserV1Opts{
			Db: databaseConnection,

			Email:    email,
			Password: password,
			Type:     models.TypeUser,
		}); err != nil {
			return fmt.Errorf("failed to create user: %s", err)
		}

		user, err := models.GetUserV1(models.GetUserV1Opts{
			Db: databaseConnection,

			Email: &email,
		})
		if err != nil {
			return fmt.Errorf("failed to retrieve user: %s", err)
		}

		user, err = models.VerifyUserV1(models.VerifyUserV1Opts{
			Db: databaseConnection,

			VerificationCode: user.EmailVerificationCode,
		})
		if err != nil {
			return fmt.Errorf("failed to verify user: %s", err)
		}

		logrus.Infof("created user[%s] with email[%s] and password[%s]", *user.Id, user.Email, password)

		if viper.GetBool("with-mfa") {
			totpSeed, err := auth.CreateTotpSeed("opsicle", user.Email)
			if err != nil {
				return fmt.Errorf("failed to create totp seed: %s", err)
			}
			userMfa, err := models.CreateUserMfaV1(models.CreateUserMfaV1Opts{
				Db: databaseConnection,

				Secret: &totpSeed,
				UserId: *user.Id,
				Type:   models.MfaTypeTotp,
			})
			if err != nil {
				return fmt.Errorf("failed to create mfa for user: %s", err)
			}
			if err := models.VerifyUserMfaV1(models.VerifyUserMfaV1Opts{
				Db: databaseConnection,

				Id: userMfa.Id,
			}); err != nil {
				return fmt.Errorf("failed to verify mfa for user: %s", err)
			}
			logrus.Infof("added mfa with seed[%s]", totpSeed)
		}

		logrus.Infof("login with:\n```\ngo run . login --email '%s' --password '%s'\n# or\nopsicle login --email '%s' --password '%s'\n```", user.Email, password, user.Email, password)

		return nil
	},
}
