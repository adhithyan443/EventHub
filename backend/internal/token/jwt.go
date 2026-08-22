package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret: []byte(secret),
	}
}

func (s *JWTService) GenerateAccessToken(
	userID uuid.UUID,
	role string,
) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"userId": userID.String(),
		"role":   role,
		"iat":    now.Unix(),
		"exp":    now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}

func (s *JWTService) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return s.secret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
