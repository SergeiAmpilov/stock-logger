package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service handles authentication logic
type Service struct {
	Password  string
	JwtSecret string
}

// NewService creates a new auth service
func NewService(passwd string, jwt string) *Service {
	return &Service{
		Password:  passwd,
		JwtSecret: jwt,
	}
}

// Authenticate validates the provided password against the configured password
func (s *Service) Authenticate(password string) (string, error) {
	if password != s.Password {
		return "", errors.New("invalid password")
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	})

	tokenString, err := token.SignedString([]byte(s.JwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
