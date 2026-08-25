package domain

type EmailService interface {
	SendOTP(to string, otp string) error
	SendPasswordResetEmail(to string, token string) error
}
