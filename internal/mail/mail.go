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
	SendPasswordReset(toEmail, resetCode string) error
}

type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendPasswordReset(toEmail, resetCode string) error {
	if s.cfg.Host == "" {
		// Mock sender if not configured
		fmt.Printf("MOCK EMAIL: Would have sent reset code %s to %s\n", resetCode, toEmail)
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	subject := "Reset your Ledger Password"
	body := fmt.Sprintf("Hello,\n\nYou requested a password reset for your Ledger account.\n\nPlease use the following temporary code to create a new password:\n\n%s\n\nThis code will expire in 1 hour. If you did not request this, please ignore this email.\n\nThanks,\nThe Ledger Team", resetCode)

	msg := []byte("From: " + s.cfg.From + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body)

	addr := s.cfg.Host + ":" + s.cfg.Port
	return smtp.SendMail(addr, auth, s.cfg.From, []string{toEmail}, msg)
}
