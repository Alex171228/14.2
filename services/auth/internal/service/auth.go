package service

import (
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService struct {
	validTokens map[string]string
}

func NewAuthService() *AuthService {
	return &AuthService{
		validTokens: map[string]string{
			"demo-token": "student",
		},
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type VerifyResponse struct {
	Valid   bool   `json:"valid"`
	Subject string `json:"subject,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *AuthService) Login(username, password string) (*LoginResponse, error) {
	if username == "student" && password == "student" {
		return &LoginResponse{
			AccessToken: "demo-token",
			TokenType:   "Bearer",
		}, nil
	}
	return nil, ErrInvalidCredentials
}

func (s *AuthService) Verify(token string) (*VerifyResponse, error) {
	if subject, ok := s.validTokens[token]; ok {
		return &VerifyResponse{
			Valid:   true,
			Subject: subject,
		}, nil
	}
	return &VerifyResponse{
		Valid: false,
		Error: "unauthorized",
	}, ErrInvalidToken
}
