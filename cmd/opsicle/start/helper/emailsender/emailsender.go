package emailsender

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"opsicle/internal/cli"

	_ "embed"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed template.html
var templateEmailBody string

//go:embed image.png
var imageBytes []byte

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

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Headers
		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", senderName, senderEmail)
		headers["To"] = fmt.Sprintf("%s <%s>", receiverName, receiverEmail)
		headers["Subject"] = "Test email from Opsicle notifications"
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "multipart/related; boundary=" + writer.Boundary()

		for k, v := range headers {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
		fmt.Fprint(&buf, "\r\n")

		// HTML Part
		htmlPart, _ := writer.CreatePart(map[string][]string{
			"Content-Type":              {"text/html; charset=UTF-8"},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		qp := quotedprintable.NewWriter(htmlPart)
		qp.Write([]byte(templateEmailBody))
		qp.Close()

		// Image Part
		imageHeader := make(textproto.MIMEHeader)
		imageHeader.Set("Content-Type", "image/png")
		imageHeader.Set("Content-Transfer-Encoding", "base64")
		imageHeader.Set("Content-ID", "<image.png>")
		imageHeader.Set("Content-Disposition", "inline; filename=\"image.png\"")
		imagePart, _ := writer.CreatePart(imageHeader)
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(imageBytes)))
		base64.StdEncoding.Encode(encoded, imageBytes)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			imagePart.Write(encoded[i:end])
			imagePart.Write([]byte("\r\n"))
		}

		writer.Close()

		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			imagePart.Write(encoded[i:end])
			imagePart.Write([]byte("\r\n"))
		}

		// Compose full message with MIME headers
		auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
		if err := smtp.SendMail(
			smtpAddr,
			auth,
			senderEmail,
			[]string{receiverEmail},
			buf.Bytes(),
		); err != nil {
			return fmt.Errorf("failed to send email: %s", err)
		}
		logrus.Infof("email sent successfully to address[%s]", receiverEmail)
		return nil
	},
}
