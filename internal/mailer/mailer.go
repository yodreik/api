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

func (m *Mailer) SendRecoveryEmail(recepient string, token string) error {
	basepath := "http://localhost:3000"
	body := fmt.Sprintf(`
		<html>
		<body>
			<p>Click <a href="%s/auth/password/reset?token=%s">here</a> to reset your password!</p>
            <p><b>Ignore this email if you didn't request a password reset</b></p>
		</body>
		</html>
	`, basepath, token)

	return m.Send(recepient, "welnex: Password reset", body)
}

func (m *Mailer) Send(recepient string, subject string, body string) error {
	auth := smtp.PlainAuth("", m.config.Address, m.config.Password, m.config.SMTP.Address)

	to := []string{recepient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		recepient, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%s", m.config.SMTP.Address, m.config.SMTP.Port), auth, m.config.Address, to, msg)
}
