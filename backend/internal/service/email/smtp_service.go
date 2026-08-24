package email

import (
	"fmt"
	"net/smtp"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
)

type SMTPEmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPEmailService(
	host string,
	port string,
	username string,
	password string,
	from string,
) domain.EmailService {
	return &SMTPEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPEmailService) SendOTP(to string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)

	subject := "Your EventHub verification code"

	body := fmt.Sprintf(
		"Your EventHub verification code is: %s\n\nThis code will expire in 10 minutes.",
		otp,
	)

	message := []byte(
		fmt.Sprintf(
			"From: %s\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"\r\n"+
				"%s",
			s.from,
			to,
			subject,
			body,
		),
	)

	address := s.host + ":" + s.port

	if err := smtp.SendMail(
		address,
		auth,
		s.from,
		[]string{to},
		message,
	); err != nil {
		return err
	}

	return nil
}
