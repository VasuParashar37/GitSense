package gitsense

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// createHTTPClient creates an HTTP client with the specified timeout
func CreateHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// createGitHubRequest creates an HTTP request with GitHub authorization header
func CreateGitHubRequest(method, url, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

// sendErrorResponse sends an HTTP error response with consistent formatting
func SendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	http.Error(w, message, statusCode)
}

// sendJSONError sends a JSON-formatted error response
func SendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}

// validateRepoParam validates the repo query parameter
func ValidateRepoParam(r *http.Request) (string, error) {
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		return "", fmt.Errorf("missing required parameter: repo")
	}
	if len(repo) > 200 {
		return "", fmt.Errorf("repo parameter too long (max 200 characters)")
	}
	return repo, nil
}

// validateLimitParam validates and returns the limit query parameter
func ValidateLimitParam(r *http.Request) (int, error) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return DefaultCommitLimit, nil
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, fmt.Errorf("invalid limit parameter: must be a number")
	}

	if limit < 1 {
		return 0, fmt.Errorf("limit must be at least 1")
	}

	if limit > MaxCommitLimit {
		return 0, fmt.Errorf("limit exceeds maximum of %d", MaxCommitLimit)
	}

	return limit, nil
}

// isFileActive determines if a file is active based on days since last modification
func IsFileActive(daysSinceModified float64) bool {
	return daysSinceModified <= float64(ActiveThreshold)
}

// isFileStable determines if a file is stable based on days since last modification
func IsFileStable(daysSinceModified float64) bool {
	return daysSinceModified > float64(ActiveThreshold) && daysSinceModified <= float64(StableThreshold)
}

// isFileInactive determines if a file is inactive based on days since last modification
func IsFileInactive(daysSinceModified float64) bool {
	return daysSinceModified > float64(InactiveThreshold)
}

// setCORSHeaders sets CORS headers for cross-origin requests
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
