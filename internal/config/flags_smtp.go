package config

import "opsicle/internal/cli"

const (
	SmtpUsername = "smtp-username"
	SmtpPassword = "smtp-password"
	SmtpHostname = "smtp-hostname"
	SmtpPort     = "smtp-port"
)

func GetSmtpFlags() cli.Flags {
	return cli.Flags{
		{
			Name:         SmtpUsername,
			DefaultValue: "noreply@notification.opsicle.io",
			Usage:        "defines the smtp server user's email address",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         SmtpPassword,
			DefaultValue: "",
			Usage:        "defines the smtp server user's password",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         SmtpHostname,
			DefaultValue: "smtp.eu.mailgun.org",
			Usage:        "defines the smtp server's hostname",
			Type:         cli.FlagTypeString,
		},
		{
			Name:         SmtpPort,
			DefaultValue: 587,
			Usage:        "defines the smtp server's port",
			Type:         cli.FlagTypeInteger,
		},
	}
}
