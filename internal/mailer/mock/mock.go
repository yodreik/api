package mock

type MockMailer struct {
	SentEmails []string
}

func New() *MockMailer {
	return &MockMailer{
		SentEmails: make([]string, 0),
	}
}

func (mm *MockMailer) SendRecoveryEmail(recepient string, token string) error {
	mm.SentEmails = append(mm.SentEmails, recepient)
	return nil
}

func (mm *MockMailer) SendConfirmationEmail(recepient string, token string) error {
	mm.SentEmails = append(mm.SentEmails, recepient)
	return nil
}

func (mm *MockMailer) SendSecurityEmail(recepient string, token string) error {
	mm.SentEmails = append(mm.SentEmails, recepient)
	return nil
}

func (mm *MockMailer) Send(recepient string, subject string, body string) error {
	mm.SentEmails = append(mm.SentEmails, recepient)
	return nil
}
