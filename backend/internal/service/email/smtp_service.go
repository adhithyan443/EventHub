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

	subject := "EventHub - Verify Your Email Address"

	body := fmt.Sprintf(`Hello,

Thank you for registering with EventHub!

To complete your registration and verify your email address, please use the following verification code:

%s

This verification code is valid for 10 minutes.

For your security, please do not share this code with anyone. EventHub will never ask you for your verification code.

If you did not create an EventHub account, you can safely ignore this email.

Best regards,
The EventHub Team
`, otp)

	message := []byte(
		fmt.Sprintf(
			"From: EventHub <%s>\r\n"+
				"To: %s\r\n"+
				"Subject: %s\r\n"+
				"MIME-Version: 1.0\r\n"+
				"Content-Type: text/plain; charset=UTF-8\r\n"+
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
