package notification

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"strconv"

	"github.com/rhyzoma/locky/internal/config"
)

type SMTPMailer struct {
	host      string
	port      int
	username  string
	password  string
	from      string
	verifyURL string
}

func NewSMTPMailer(cfg config.MailConfig) (*SMTPMailer, error) {
	if cfg.Host == "" || cfg.Port <= 0 || cfg.From == "" || cfg.VerifyURL == "" {
		return nil, errors.New("smtp mailer: host, port, from, and verify url are required")
	}
	return &SMTPMailer{
		host: cfg.Host, port: cfg.Port, username: cfg.Username, password: cfg.Password,
		from: cfg.From, verifyURL: cfg.VerifyURL,
	}, nil
}

func (m *SMTPMailer) SendVerification(ctx context.Context, email, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your email\r\n\r\nVerify your email: %s?token=%s\r\n", m.from, email, m.verifyURL, token)
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}
	if err := smtp.SendMail(m.host+":"+strconv.Itoa(m.port), auth, m.from, []string{email}, []byte(message)); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

func (m *SMTPMailer) SendPasswordReset(context.Context, string, string) error {
	return errors.New("password reset mail is not implemented")
}
