package emailsender

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common"
	"opsicle/internal/common/images"
	"opsicle/internal/email"
	"time"

	_ "embed"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed template.html
var templateEmailBody []byte

var flags cli.Flags = cli.Flags{
	{
		Name:         "receiver-name",
		Short:        'R',
		DefaultValue: "You",
		Usage:        "defines the receiver's name",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "receiver-email",
		Short:        'r',
		DefaultValue: "hello@opsicle.io",
		Usage:        "defines the receiver's email address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "sender-name",
		Short:        'S',
		DefaultValue: "Opsicle Notifications",
		Usage:        "defines the sender's name",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-username",
		Short:        's',
		DefaultValue: "noreply@notification.opsicle.io",
		Usage:        "defines the sender's email address",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-password",
		Short:        'p',
		DefaultValue: "",
		Usage:        "defines the sender's password",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-hostname",
		Short:        'H',
		DefaultValue: "smtp.eu.mailgun.org",
		Usage:        "defines the smtp server's hostname",
		Type:         cli.FlagTypeString,
	},
	{
		Name:         "smtp-port",
		Short:        'P',
		DefaultValue: 587,
		Usage:        "defines the smtp server's port",
		Type:         cli.FlagTypeInteger,
	},
}

func init() {
	flags.AddToCommand(Command)
}

var Command = &cobra.Command{
	Use:     "emailsender",
	Aliases: []string{"email"},
	Short:   "Sends a test email given SMTP credentials",
	PreRun: func(cmd *cobra.Command, args []string) {
		flags.BindViper(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// SMTP settings
		smtpHost := viper.GetString("smtp-hostname")
		smtpPort := viper.GetInt("smtp-port")
		smtpAddr := fmt.Sprintf("%s:%v", smtpHost, smtpPort)
		logrus.Infof("sending via smtp[%s]", smtpAddr)

		smtpUsername := viper.GetString("smtp-username")
		smtpPassword := viper.GetString("smtp-password")
		logrus.Infof("using user[%s]", smtpUsername)

		receiverName := viper.GetString("receiver-name")
		receiverEmail := viper.GetString("receiver-email")
		senderEmail := smtpUsername
		senderName := viper.GetString("sender-name")
		logrus.Infof("sending message from address[%s] to address[%s]...", senderEmail, receiverEmail)

		serviceLogs := make(chan common.ServiceLog, 64)
		common.StartServiceLogLoop(serviceLogs)

		opsicleCatMimeType, opsicleCatData := images.GetOpsicleCat()
		if err := email.SendSmtp(email.SendSmtpOpts{
			ServiceLogs: serviceLogs,

			To: []email.User{
				{
					Address: receiverEmail,
					Name:    receiverName,
				},
			},
			Sender: email.User{
				Address: senderEmail,
				Name:    senderName,
			},
			Smtp: email.SmtpConfig{
				Hostname: smtpHost,
				Port:     smtpPort,
				Username: smtpUsername,
				Password: smtpPassword,
			},
			Message: email.Message{
				Body:  templateEmailBody,
				Title: "Test email from Opsicle Notifications",
				Images: map[string]email.MessageAttachment{
					"image.png": {
						Type: opsicleCatMimeType,
						Data: opsicleCatData,
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("failed to trigger email: %s", err)
		}

		<-time.After(500 * time.Millisecond)
		return nil
	},
}
