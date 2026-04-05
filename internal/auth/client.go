package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kiev/kernus/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrSessionExpired     = errors.New("session has expired, please login again")
	ErrServerUnreachable  = errors.New("cannot reach server")
	ErrOAuthOnlyAccount   = errors.New("oauth_only_account")
)

type AuthClient interface {
	Login(ctx context.Context, username, password string) (*models.Session, error)
	Logout(ctx context.Context, token string) error
	GetProfile(ctx context.Context, token string) (*models.User, error)
	ValidateToken(ctx context.Context, token string) (bool, error)
}

type LocalClient struct{}

func NewDefaultClient() AuthClient {
	serverURL := strings.TrimSpace(os.Getenv("KERNUS_SERVER_URL"))
	if serverURL == "" {
		return NewLocalClient()
	}
	return NewHTTPClient(serverURL)
}

func NewLocalClient() *LocalClient {
	return &LocalClient{}
}

func (c *LocalClient) Login(ctx context.Context, username, password string) (*models.Session, error) {
	return &models.Session{
		Token:     "local-" + username,
		Username:  username,
		UserID:    "local",
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

func (c *LocalClient) Logout(ctx context.Context, token string) error {
	return nil
}

func (c *LocalClient) GetProfile(ctx context.Context, token string) (*models.User, error) {
	if strings.HasPrefix(token, "local-") {
		name := strings.TrimPrefix(token, "local-")
		return &models.User{
			ID:       "local",
			Username: name,
			Email:    name,
			Role:     models.RoleAdmin,
			Groups:   []string{},
		}, nil
	}
	return nil, fmt.Errorf("local profile unavailable for this token")
}

func (c *LocalClient) ValidateToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}
