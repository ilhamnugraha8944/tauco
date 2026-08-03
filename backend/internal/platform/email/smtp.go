package email

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
)

type SMTPConfig struct {
	Host string
	Port int
	From string
	To   string
}

type SMTPSender struct {
	config SMTPConfig
	send   func(string, smtp.Auth, string, []string, []byte) error
}

func NewSMTPSender(config SMTPConfig) (*SMTPSender, error) {
	if config.Host == "" || config.Port < 1 || config.Port > 65535 ||
		config.From == "" || config.To == "" ||
		strings.ContainsAny(config.From+config.To, "\r\n") {
		return nil, errors.New("invalid SMTP configuration")
	}
	return &SMTPSender{config: config, send: smtp.SendMail}, nil
}

func (sender *SMTPSender) SendContactNotification(
	ctx context.Context,
	notification contactapp.Notification,
	reference string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.ContainsAny(reference, "\r\n") {
		return errors.New("invalid email idempotency reference")
	}
	message := []byte(
		"From: " + sender.config.From + "\r\n" +
			"To: " + sender.config.To + "\r\n" +
			"Subject: Pesan baru Tauco Cap Badak\r\n" +
			"X-Tauco-Job-ID: " + reference + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
			"Nama: " + notification.Name + "\n" +
			"Email: " + notification.Email + "\n" +
			"Subjek: " + notification.Subject + "\n\n" +
			notification.Message + "\n",
	)
	address := net.JoinHostPort(sender.config.Host, strconv.Itoa(sender.config.Port))
	if err := sender.send(address, nil, sender.config.From, []string{sender.config.To}, message); err != nil {
		return fmt.Errorf("SMTP send: %w", err)
	}
	return nil
}
