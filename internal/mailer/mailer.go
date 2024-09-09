package mailer

import (
	"api/internal/config"
	"fmt"
	"net/smtp"
)

const basepath = "http://localhost:3000"

type Mailer interface {
	SendRecoveryEmail(recepient string, token string) error
	SendConfirmationEmail(recepient string, token string) error
	Send(recepient string, subject string, body string) error
}

type Sender struct {
	config config.Mail
}

func New(c config.Mail) *Sender {
	return &Sender{
		config: c,
	}
}

func (s *Sender) SendRecoveryEmail(recepient string, token string) error {
	body := fmt.Sprintf(`
		<html>
		<body>
			<p>Click <a href="%s/auth/password/reset?token=%s">here</a> to reset your password!</p>
            <p><b>Ignore this email if you didn't request a password reset</b></p>
		</body>
		</html>
	`, basepath, token)

	return s.Send(recepient, "welnex: Password reset", body)
}

func (s *Sender) SendConfirmationEmail(recepient string, token string) error {
	body := fmt.Sprintf(`
		<html>
		<body>
			<p>Click <a href="%s/auth/confirm?token=%s">here</a> to verify your account!</p>
			<p>This link will be available only for 48h!</p>
		</body>
		</html>
	`, basepath, token)

	return s.Send(recepient, "welnex: Account confirmation", body)
}

func (s *Sender) Send(recepient string, subject string, body string) error {
	auth := smtp.PlainAuth("", s.config.Address, s.config.Password, s.config.SMTP.Address)

	to := []string{recepient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		recepient, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%s", s.config.SMTP.Address, s.config.SMTP.Port), auth, s.config.Address, to, msg)
}
