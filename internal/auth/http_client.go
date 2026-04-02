package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kiev/kernus/internal/models"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	trimmed := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	return &HTTPClient{
		baseURL: trimmed,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPClient) Login(ctx context.Context, username, password string) (*models.Session, error) {
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type loginResponse struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}

	body, err := json.Marshal(loginRequest{Email: username, Password: password})
	if err != nil {
		return nil, fmt.Errorf("cannot encode login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		var apiErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Code == "OAUTH_ONLY_ACCOUNT" {
			return nil, ErrOAuthOnlyAccount
		}
		return nil, ErrInvalidCredentials
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("login failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed loginResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("invalid login response: %w", err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("login response missing access_token")
	}

	expiresAt, expErr := ExtractExpiry(parsed.AccessToken)
	if expErr != nil {
		expiresAt = time.Now().Add(15 * time.Minute)
	}

	return &models.Session{
		Token:     parsed.AccessToken,
		Username:  parsed.User.Email,
		UserID:    parsed.User.ID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

func (c *HTTPClient) Logout(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth/logout", nil)
	if err != nil {
		return fmt.Errorf("cannot create logout request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("logout failed with status %d", resp.StatusCode)
	}
	return nil
}

func (c *HTTPClient) GetProfile(ctx context.Context, token string) (*models.User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profile request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrSessionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("profile failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var user models.User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("invalid profile response: %w", err)
	}
	return &user, nil
}

func (c *HTTPClient) ValidateToken(ctx context.Context, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/users/me", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, fmt.Errorf("validate token failed with status %d", resp.StatusCode)
}
