package auth

import "github.com/kiev/kernus/internal/models"

// UserFromStoredSession builds profile fields from the active session and optional stored record.
func UserFromStoredSession(sess *models.Session, stored *StoredSession) models.User {
	if sess == nil {
		return models.User{}
	}
	email := sess.Username
	if stored != nil && stored.Email != "" {
		email = stored.Email
	}
	return models.User{
		ID:       sess.UserID,
		Username: sess.Username,
		Email:    email,
		Role:     models.RoleViewer,
		Groups:   nil,
	}
}
