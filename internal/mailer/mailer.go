package mailer

import (
	"api/internal/config"
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
)

type ConfirmationEmailData struct {
	BasePath string
	Token    string
}

type RecoveryEmailData struct {
	BasePath string
	Token    string
}

type Mailer interface {
	SendRecoveryEmail(recepient string, token string) error
	SendConfirmationEmail(recepient string, token string) error
	Send(recepient string, subject string, body string) error
}

type Sender struct {
	config *config.Config
}

func New(c *config.Config) *Sender {
	return &Sender{
		config: c,
	}
}

func (s *Sender) SendRecoveryEmail(recepient string, token string) error {
	tmpl, err := template.ParseFiles("templates/recovery_email.html")
	if err != nil {
		return fmt.Errorf("Error parsing template: %v", err)
	}

	data := RecoveryEmailData{
		BasePath: s.config.BasePath,
		Token:    token,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}

	return s.Send(recepient, "yodreik: Password reset", buf.String())
}

func (s *Sender) SendConfirmationEmail(recepient string, token string) error {
	tmpl, err := template.ParseFiles("templates/confirmation_email.html")
	if err != nil {
		return fmt.Errorf("Error parsing template: %v", err)
	}

	data := ConfirmationEmailData{
		BasePath: s.config.BasePath,
		Token:    token,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}

	return s.Send(recepient, "yodreik: Account confirmation", buf.String())
}

func (s *Sender) Send(recepient string, subject string, body string) error {
	auth := smtp.PlainAuth("", s.config.Mail.Address, s.config.Mail.Password, s.config.Mail.SMTP.Address)

	to := []string{recepient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		recepient, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%s", s.config.Mail.SMTP.Address, s.config.Mail.SMTP.Port), auth, s.config.Mail.Address, to, msg)
}
