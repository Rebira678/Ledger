package mail

import (
	"fmt"
	"net/smtp"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Sender interface {
	SendPasswordReset(toEmail, resetLink string) error
}

type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendPasswordReset(toEmail, resetLink string) error {
	if s.cfg.Host == "" {
		// Mock sender if not configured
		fmt.Printf("MOCK EMAIL: Would have sent reset link %s to %s\n", resetLink, toEmail)
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	subject := "Reset your Ledger Password"
	body := fmt.Sprintf("Hello,\n\nYou requested a password reset for your Ledger account.\n\nPlease click the following link to create a new password:\n%s\n\nIf you did not request this, please ignore this email.\n\nThanks,\nThe Ledger Team", resetLink)

	msg := []byte("From: " + s.cfg.From + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body)

	addr := s.cfg.Host + ":" + s.cfg.Port
	return smtp.SendMail(addr, auth, s.cfg.From, []string{toEmail}, msg)
}
