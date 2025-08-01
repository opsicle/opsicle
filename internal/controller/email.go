package controller

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"opsicle/internal/email"
	"time"
)

var smtpConfig SmtpServerConfig

type SmtpServerConfig struct {
	Hostname string
	Port     int
	Username string
	Password string

	Sender email.User
}

func (c SmtpServerConfig) IsSet() bool {
	return c.Hostname != "" && c.Port > 0 && c.Username != "" && c.Password != "" && c.Sender.Address != ""
}

func (c SmtpServerConfig) VerifyConnection() error {
	addr := fmt.Sprintf("%s:%v", c.Hostname, c.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP connection failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, c.Hostname)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         c.Hostname,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	auth := smtp.PlainAuth("", c.Username, c.Password, c.Hostname)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("failed to auth with user[%s]: %w", c.Username, err)
	}

	return nil
}
