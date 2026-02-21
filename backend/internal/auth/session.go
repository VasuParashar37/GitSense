package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitsense"
	"gitsense/internal/db"
)

func ExtractSessionToken(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		if token == "" {
			return "", errors.New("missing bearer token")
		}
		return token, nil
	}

	return authHeader, nil
}

func CreateSession(userID int, githubToken string) (string, error) {
	token, err := generateSecureToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().UTC().Add(time.Duration(gitsense.SessionTTLHours) * time.Hour).Format(time.RFC3339)
	_, err = db.DB.Exec(`
		INSERT INTO sessions (user_id, session_token, github_token, expires_at, last_used_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, userID, token, githubToken, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

func ResolveGitHubToken(sessionToken string) (githubToken string, userID int, err error) {
	err = db.DB.QueryRow(`
		SELECT user_id, github_token
		FROM sessions
		WHERE session_token = ?
		  AND julianday(expires_at) > julianday('now')
	`, sessionToken).Scan(&userID, &githubToken)
	if err != nil {
		return "", 0, fmt.Errorf("invalid or expired session")
	}

	_, _ = db.DB.Exec(`
		UPDATE sessions
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE session_token = ?
	`, sessionToken)

	return githubToken, userID, nil
}

func generateSecureToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
