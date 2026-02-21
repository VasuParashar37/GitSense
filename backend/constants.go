package gitsense

import "time"

// API and GitHub Configuration
const (
	// GitHub API limits
	DefaultCommitLimit = 30
	MaxCommitLimit     = 100

	// HTTP Client timeouts
	GitHubAPITimeout = 30 * time.Second
	DefaultTimeout   = 10 * time.Second

	// Database retry configuration
	MaxDBRetries      = 5
	InitialRetryDelay = 100 * time.Millisecond

	// Activity thresholds (in days)
	ActiveThreshold   = 7
	StableThreshold   = 30
	InactiveThreshold = 30

	// Server timeouts
	ServerReadTimeout  = 15 * time.Second
	ServerWriteTimeout = 15 * time.Second
	ServerIdleTimeout  = 60 * time.Second

	// Historical snapshot window
	HistoricalSnapshotDays = 30
)
