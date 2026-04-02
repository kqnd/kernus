package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type tokenClaims struct {
	OrgID string `json:"org_id"`
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

func parseTokenClaims(token string) (*tokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt token format")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("cannot decode jwt payload: %w", err)
	}

	var claims tokenClaims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, fmt.Errorf("cannot parse jwt payload: %w", err)
	}

	return &claims, nil
}

func ExtractOrgID(token string) (string, error) {
	claims, err := parseTokenClaims(token)
	if err != nil {
		return "", err
	}
	if claims.OrgID == "" {
		return "", fmt.Errorf("org_id claim not found in access token")
	}
	return claims.OrgID, nil
}

func ExtractExpiry(token string) (time.Time, error) {
	claims, err := parseTokenClaims(token)
	if err != nil {
		return time.Time{}, err
	}
	if claims.Exp <= 0 {
		return time.Time{}, fmt.Errorf("exp claim not found in access token")
	}
	return time.Unix(claims.Exp, 0).UTC(), nil
}
