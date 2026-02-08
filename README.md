# GitSense

GitHub repository insights and analytics tool with Chrome extension integration.

## 🚀 Quick Setup

### 1. Configure Environment Variables

Copy the example environment file and add your GitHub OAuth credentials:

```bash
cp .env.example .env
```

Then edit [.env](.env) and add your credentials:
- Get your GitHub OAuth credentials from: https://github.com/settings/developers
- Fill in `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET`

### 2. Run the Backend

#### Option A: Using the automated dev script (Recommended)
```bash
./dev.sh
```

This script will:
- ✅ Automatically load environment variables from [.env](.env)
- 🚀 Start the backend server

#### Option B: Using the setup script
```bash
./setup.sh
```

#### Option C: Manual export
```bash
export GITHUB_CLIENT_ID="your_client_id"
export GITHUB_CLIENT_SECRET="your_client_secret"
cd backend && go run .
```

## 📁 Project Structure

```
GitSense/
├── backend/           # Go backend server
│   ├── main.go       # Entry point
│   ├── auth.go       # GitHub OAuth
│   ├── api.go        # API handlers
│   ├── repos.go      # Repository operations
│   └── sync.go       # Data synchronization
├── chrome-extension/ # Chrome extension
│   ├── popup.js
│   └── background.js
├── .env              # Your credentials (git-ignored)
├── .env.example      # Template for credentials
├── dev.sh            # Quick development script
└── setup.sh          # Environment setup script
```

## 🔐 Security Notes

- Never commit [.env](.env) to version control (already in [.gitignore](.gitignore))
- Keep your GitHub OAuth credentials secure
- The [.env.example](.env.example) file is safe to commit (contains no real credentials)

## 🛠️ Development

The backend runs on port 8080 by default. You can change this in [.env](.env).

### Available Endpoints
- `GET /health` - Health check
- `GET /auth/github` - GitHub OAuth login
- `GET /auth/callback/` - OAuth callback
- `GET /repos` - Get user repositories
- `POST /sync` - Sync repository data
- `GET /project/summary` - Get project summary
- `GET /history` - Get repository history
