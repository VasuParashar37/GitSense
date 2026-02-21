# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GitSense is a GitHub repository insights and analytics tool consisting of:
- **Backend**: Go server providing RESTful API with GitHub OAuth integration
- **Chrome Extension**: Browser extension for visualizing repository analytics

## Tech Stack

- **Backend**: Go 1.21, SQLite (modernc.org/sqlite), godotenv for config
- **Extension**: Chrome Extension Manifest V3, Chart.js for visualizations
- **Database**: SQLite with WAL mode enabled for better concurrency
- **Auth**: GitHub OAuth 2.0 flow with token-based authentication

## Development Commands

### Backend
```bash
# Start the backend server (from project root)
cd backend && go run .

# Install/update dependencies
cd backend && go mod tidy

# The backend runs on port 8080 by default (configurable via PORT env var)
```

### Chrome Extension
The extension must be loaded manually in Chrome:
1. Navigate to `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked" and select the `chrome-extension/` directory

## Environment Configuration

Required environment variables in `.env` file at project root:
- `GITHUB_CLIENT_ID`: OAuth app client ID from https://github.com/settings/developers
- `GITHUB_CLIENT_SECRET`: OAuth app client secret
- `PORT`: Backend server port (default: 8080)
- `EXTENSION_ORIGIN`: **Security-critical** - Origin for postMessage token delivery
  - For Chrome extension: `chrome-extension://<your-extension-id>` (find ID at chrome://extensions/)
  - For local development: `http://localhost` (default if not set)
  - **Never use `*`** - this would expose tokens to any website

The backend's `main.go` loads `.env` from parent directory (`../.env` relative to backend/).

## Architecture

### Authentication Flow
1. User clicks "Login with GitHub" in extension popup
2. Extension opens OAuth flow via `/auth/github` endpoint
3. GitHub redirects to `/auth/callback` after authorization
4. Backend exchanges code for access token, stores in `users` table
5. Token passed to extension via `/token` endpoint using `postMessage`
6. Extension stores token in `chrome.storage.local` for persistence

### Data Synchronization
- Extension triggers sync via `POST /sync?owner=X&repo=Y` with token in Authorization header
- Backend fetches commits and file data from GitHub API (limited to 30 commits per sync)
- Data stored in SQLite tables: `commits`, `file_activity`, `repo_snapshots`
- Extension polls for updates using background service worker

### Database Schema
- `users`: GitHub username, access token, timestamps
- `user_repos`: Links users to their synced repositories
- `commits`: Commit SHA, author, message, date per repository
- `file_activity`: Aggregated file change counts and last modified dates (unique per repo+filename)
- `repo_snapshots`: Historical snapshots of repository activity metrics

### API Endpoints
- `GET /health`: Health check endpoint
- `GET /auth/github`: Initiates GitHub OAuth flow
- `GET /auth/callback`: OAuth callback handler
- `GET /token?value=<token>`: Token bridge to extension (postMessage wrapper)
- `GET /repos`: Fetch authenticated user's repositories
- `POST /sync?owner=X&repo=Y`: Sync repository data from GitHub (requires Authorization header)
- `GET /project/summary`: Get aggregated project statistics (includes activity score)
- `GET /history?repo=X`: Repository activity history (returns snapshots with integer scores)
- `GET /commits?repo=X&limit=N`: Commit data with SHA, author, message, date (optional limit)
- `GET /files?repo=X`: File activity metrics with status categorization
- `GET /commits-per-day?repo=X`: Aggregated commits by date (last 30 days)
- `GET /file-breakdown?repo=X`: Files categorized as most_modified, inactive, frequently_updated
- `GET /contributor-distribution?repo=X`: Commit counts per author
- `GET /dashboard?repo=X`: Full-screen HTML dashboard with interactive analytics

### Extension Architecture
- `popup.js`: Main UI logic, handles auth state, repository selection, chart rendering
- `background.js`: Service worker for periodic sync and notifications
- `popup.html`: Extension popup UI with Chart.js integration
- Token persists across browser sessions via `chrome.storage.local`

### Dashboard Features
The full dashboard (`dashboard.html`, ~910 lines) provides advanced analytics:
- **Interactive Timeline**: Time-series commit visualization with Chart.js and chartjs-plugin-zoom
  - Intelligent time bucketing (hour/day/week/month based on data range)
  - Commit frequency calculation using 1-hour sliding window
  - Scroll to zoom, drag to pan, reset zoom functionality
- **Commit History Table**: Paginated table (20 commits per page) with SHA, author, message, date
- **File Activity Breakdown**: Three-tab interface for most modified, inactive (>30 days), frequently updated files
- **Activity Trends**: Commits per day bar chart and contributor distribution pie chart
- **Export Functionality**: Download data as JSON or CSV

### Activity Score Calculation
Core metric representing percentage of actively developed files:
1. **File Categorization** (by days since last modification):
   - Active: ≤7 days
   - Stable: 7-30 days
   - Inactive: >30 days
2. **Score Formula**: `(active_files / total_files) × 100` (rounded to integer)
3. **State Interpretation**:
   - HIGH ACTIVITY: score > 50
   - EVOLVING: 25 < score ≤ 50
   - STABLE: score ≤ 25
4. **Storage**: Snapshots created on sync (one per day max) with `CURRENT_TIMESTAMP`

## Key Files

### Backend Core
- `main.go`: HTTP server setup, route handlers, environment loading
- `auth.go`: GitHub OAuth implementation
- `github.go`: GitHub API client for fetching commits and file data
- `sync.go`: Synchronization logic, snapshot creation, activity score calculation
- `db.go`: SQLite initialization, schema definitions, WAL mode configuration
- `api.go`: Dashboard API endpoints (commits-per-day, file-breakdown, contributor-distribution)
- `commits.go`: Commit fetching with optional limit parameter
- `repos.go`: Repository listing for authenticated users
- `models.go`: Data models (ProjectSummary, etc.)
- `dashboard.html`: Full-screen analytics dashboard served by backend

### Extension
- `popup.js`: Extension UI controller (~450 lines)
- `background.js`: Service worker for background tasks
- `manifest.json`: Extension permissions and configuration

## Important Notes

- The backend expects `.env` to be in the project root, not inside `backend/`
- SQLite database file (`gitsense.db`) is created in `backend/` directory
- GitHub API requests are limited to 30 commits per sync to avoid timeouts
- Extension requires `storage`, `identity`, `tabs`, and `notifications` permissions
- Backend supports both localhost (dev) and render.com (production) origins in extension manifest
- **Date Format**: All timestamps use RFC3339 format (`2026-02-17T10:30:00Z`)
  - Use `time.Parse(time.RFC3339, dateString)` when parsing dates
  - SQLite stores as `DATETIME`, converts automatically
- **File Updates**: `last_modified` only updates with newer dates using CASE statement in `github.go`
- **Snapshot Logic**: Maximum one snapshot per day per repo (checked before creation)
- **Activity Scores**: Returned as integers (rounded) in all API responses

## Database Concurrency

The SQLite database uses these pragmas for better concurrent access:
- `PRAGMA journal_mode=WAL` - Write-Ahead Logging for concurrent reads during writes
- `PRAGMA busy_timeout=5000` - Wait up to 5 seconds if database is locked
- `PRAGMA synchronous=NORMAL` - Balance between safety and performance

## Common Workflows

### Adding a New API Endpoint
1. Add route handler in `main.go` using `http.HandleFunc`
2. Implement handler function in appropriate file (`api.go`, `repos.go`, etc.)
3. If querying GitHub API, add logic to `github.go`
4. If storing data, add database operations to `db.go` or relevant file
5. Update extension's `popup.js` or `background.js` to call the new endpoint

### Modifying Database Schema
1. Update table creation SQL in `db.go` `InitDB()` function
2. Consider migration strategy for existing databases (currently no migration system)
3. SQLite will only create tables if they don't exist (`CREATE TABLE IF NOT EXISTS`)

### Debugging Authentication Issues
- Check backend logs for token presence: `🔑 GITHUB_CLIENT_ID` log lines in `main.go`
- Verify extension has stored token: check `chrome.storage.local` in extension DevTools
- Ensure Authorization header is passed: format is `Bearer <token>` not just `<token>`

### Debugging Dashboard Issues
- **Empty Data**: Verify API endpoints return JSON (check browser Network tab)
- **Chart Not Rendering**: Check console for Chart.js errors, ensure date adapter is loaded
- **Decimal Scores**: Activity scores should be integers; check backend rounding logic in `sync.go` and `api.go`
- **Active Files Count Zero**: Verify date parsing uses `time.RFC3339` not custom format
- **Browser Cache**: Hard refresh (Cmd+Shift+R / Ctrl+Shift+R) after backend changes
