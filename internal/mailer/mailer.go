package mailer

import (
	"api/internal/config"
	"fmt"
	"net/smtp"
)

type Mailer struct {
	config config.Mail
}

func New(c config.Mail) *Mailer {
	return &Mailer{
		config: c,
	}
}

func (m *Mailer) Send(recepient string, subject string, body string) error {
	auth := smtp.PlainAuth("", m.config.Address, m.config.Password, m.config.SMTP.Address)

	to := []string{recepient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", recepient, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%s", m.config.SMTP.Address, m.config.SMTP.Port), auth, m.config.Address, to, msg)
}