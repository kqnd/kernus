package models

import (
	"fmt"
	"time"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

func (r Role) CanManageContainers() bool {
	return r == RoleAdmin || r == RoleOperator
}

func (r Role) CanRemove() bool {
	return r == RoleAdmin
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Groups    []string  `json:"groups"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

type Session struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) TimeUntilExpiry() time.Duration {
	return time.Until(s.ExpiresAt)
}

func (s *Session) FormatExpiry() string {
	d := s.TimeUntilExpiry()
	if d <= 0 {
		return "expired"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("expires in %dh%dm", hours, minutes)
}

func MockUser() User {
	return User{
		ID:        "usr-001",
		Username:  "john_doe",
		Email:     "john@acme.com",
		Role:      RoleAdmin,
		Groups:    []string{"backend", "web", "monitoring"},
		CreatedAt: time.Now().Add(-90 * 24 * time.Hour),
		LastLogin: time.Now().Add(-2 * time.Hour),
	}
}
