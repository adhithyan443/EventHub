package domain

type GoogleUser struct {
	Email     string
	Name      string
	GoogleID  string
	AvatarURL string
}

type GoogleOAuthService interface {
	GetAuthURL(state string) string
	GetUser(code string) (*GoogleUser, error)
}
