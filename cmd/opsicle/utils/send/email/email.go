package email

import (
	"fmt"
	"opsicle/internal/cli"
	"opsicle/internal/common/images"
	"opsicle/internal/config"
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
}.Append(config.GetSmtpFlags())

var Command = cli.NewCommand(cli.CommandOpts{
	Name:  "utils.send.email",
	Flags: flags,
	Use:   "email",
	Short: "Sends a test email given SMTP credentials",
	Run: func(cmd *cobra.Command, opts *cli.Command, args []string) error {
		serviceLogs := opts.GetServiceLogs()

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
			return fmt.Errorf("failed to trigger email: %w", err)
		}

		<-time.After(500 * time.Millisecond)
		return nil
	},
})
