package auth

import (
	"context"
	"time"

	"github.com/kiev/kernus/internal/models"
)

type MockClient struct {
	Users map[string]string
}

func NewMockClient() *MockClient {
	return &MockClient{
		Users: map[string]string{
			"admin": "admin123",
			"user":  "user123",
		},
	}
}

func (m *MockClient) Login(ctx context.Context, username, password string) (*models.Session, error) {
	expected, exists := m.Users[username]
	if !exists || expected != password {
		return nil, ErrInvalidCredentials
	}

	return &models.Session{
		Token:     "mock-token-" + username,
		Username:  username,
		UserID:    "user-001",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func (m *MockClient) Logout(ctx context.Context, token string) error {
	return nil
}

func (m *MockClient) GetProfile(ctx context.Context, token string) (*models.User, error) {
	u := models.MockUser()
	return &u, nil
}

func (m *MockClient) ValidateToken(ctx context.Context, token string) (bool, error) {
	return token != "", nil
}
