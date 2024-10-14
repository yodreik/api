package mailer

import (
	"api/internal/config"
	"fmt"
	"net/smtp"
)

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
	body := fmt.Sprintf(`
		<html>
		<body>
			<p>Click <a href="%s/auth/password/reset?token=%s">here</a> to reset your password!</p>
            <p><b>Ignore this email if you didn't request a password reset</b></p>
		</body>
		</html>
	`, s.config.BasePath, token)

	return s.Send(recepient, "dreik: Password reset", body)
}

func (s *Sender) SendConfirmationEmail(recepient string, token string) error {
	body := fmt.Sprintf(`
		<html>
		<body>
			<p>Click <a href="%s/auth/confirm?token=%s">here</a> to verify your account!</p>
			<p>This link will be available only for 48h!</p>
		</body>
		</html>
	`, s.config.BasePath, token)

	return s.Send(recepient, "dreik: Account confirmation", body)
}

func (s *Sender) Send(recepient string, subject string, body string) error {
	auth := smtp.PlainAuth("", s.config.Mail.Address, s.config.Mail.Password, s.config.Mail.SMTP.Address)

	to := []string{recepient}
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		recepient, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%s", s.config.Mail.SMTP.Address, s.config.Mail.SMTP.Port), auth, s.config.Mail.Address, to, msg)
}
