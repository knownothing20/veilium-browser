package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

func GenerateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate API token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func ValidateToken(token string) error {
	if len(strings.TrimSpace(token)) < 24 {
		return fmt.Errorf("API token must contain at least 24 characters")
	}
	return nil
}
