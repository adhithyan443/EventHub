package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/adhithyan443/EventHub/backend/internal/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleOAuthService struct {
	oauthConfig *oauth2.Config
}

func NewGoogleOAuthService(
	clientID string,
	clientSecret string,
	redirectURL string,
) *GoogleOAuthService {
	return &GoogleOAuthService{
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
		},
	}
}

func (s *GoogleOAuthService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
	)
}

func (s *GoogleOAuthService) GetUser(code string) (*domain.GoogleUser, error) {
	token, err := s.oauthConfig.Exchange(
		context.Background(),
		code,
	)
	if err != nil {
		return nil, fmt.Errorf("exchange google authorization code: %w", err)
	}

	client := s.oauthConfig.Client(
		context.Background(),
		token,
	)

	response, err := client.Get(
		"https://www.googleapis.com/oauth2/v2/userinfo",
	)
	if err != nil {
		return nil, fmt.Errorf("get google user info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"google user info returned status: %s",
			response.Status,
		)
	}

	var googleProfile struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(response.Body).Decode(&googleProfile); err != nil {
		return nil, fmt.Errorf("decode google user info: %w", err)
	}

	return &domain.GoogleUser{
		GoogleID:  googleProfile.ID,
		Email:     googleProfile.Email,
		Name:      googleProfile.Name,
		AvatarURL: googleProfile.Picture,
	}, nil
}
