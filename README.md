# Granola CLI Go

A practical Go CLI for Granola meetings, Gmail, Calendar, and semantic search.

[![CI](https://github.com/ShaneOxM/granola-cli-go/actions/workflows/ci.yml/badge.svg)](https://github.com/ShaneOxM/granola-cli-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

## Important

This is an unofficial, independent Granola CLI implementation. It is not
affiliated with, endorsed by, or connected to Granola Labs, Inc. It uses your
local Granola desktop credentials and the publicly reachable Granola APIs to
work with your own meeting data.

## Note

This CLI is actively used and validated on macOS. Linux support exists for core
Go workflows, but the most battle-tested path is macOS with the Granola desktop
app installed.

## Features

- **Meetings**: List, view, transcript, and notes for all your meetings
- **Workspaces & Folders**: Organize meetings by workspace and folder
- **Gmail Integration**: Search and retrieve Gmail messages with attendee-aware queries
- **Calendar Integration**: List events and enrich meeting records with calendar metadata
- **Context Management**: Attach contextual notes and track progression
- **AI Helpers**: Summaries, actions, key takeaways, and semantic search

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Google Setup](#google-setup)
- [Usage](#usage)
  - [Authentication](#authentication)
  - [Meetings](#meetings)
  - [Workspaces & Folders](#workspaces--folders)
  - [Gmail](#gmail)
  - [Calendar](#calendar)
  - [Embeddings & Semantic Search](#embeddings--semantic-search)
  - [Agent Workflows](#agent-workflows)
  - [Context](#context)
  - [Configuration](#configuration-1)
- [Troubleshooting](#troubleshooting)
- [Acknowledgments](#acknowledgments)

## Installation

### Prerequisites

- Go 1.21 or later if building from source
- Granola desktop app installed for meeting sync/auth import

### npm / npx

Use npm when you want the packaged CLI experience without manually copying
binaries.

```bash
# One-off usage with npx
npx granola-cli-go meeting list --limit 5

# Global install
npm install -g granola-cli-go
granola meeting list --limit 5
```

The npm package ships prebuilt binaries for supported platforms, so users do
not need a local Go toolchain for the npm / npx path.

### Build from Source

```bash
git clone https://github.com/ShaneOxM/granola-cli-go.git
cd granola-cli-go

# Build + install as the active `granola` command
./scripts/install.sh
```

The installer builds the Go binary, replaces any existing `granola` shim, and
backs the previous command up as `granola-node`.

### Using Pre-built Binary

Download the latest binary from releases and place it in your PATH:

```bash
# macOS
cp bin/granola /usr/local/bin/

# Linux
sudo cp bin/granola /usr/local/bin/
```

## Quick Start

If you already have the Granola desktop app installed locally:

```bash
# Import Granola desktop credentials
granola auth login

# Verify meetings are available
granola meeting list --limit 5

# Set up Gmail/Calendar access (recommended)
granola gmail setup-oauth
granola gmail login

# Build semantic search coverage
granola embedding backfill

# Use the CLI
granola search "meeting summary"
granola gmail person someone@example.com --max=10
granola calendar enrich-meetings
```

## Configuration

Most users only need configuration for Google setup or optional AI summary settings.

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GRANOLA_CONFIG_PATH` | No | Custom config file path |
| `GRANOLA_OAUTH_CLIENT_ID` | No | Google OAuth client ID (optional if saved in config) |
| `GRANOLA_OAUTH_CLIENT_SECRET` | No | Google OAuth client secret (optional if saved in config) |
| `GOOGLE_WORKSPACE_CLI_TOKEN` | No | Explicit bearer token override for Gmail/Calendar |
| `GRANOLA_DEBUG` | No | Enable debug logging (1 = on) |
| `GRANOLA_INFERENCE_TIMEOUT` | No | Inference request timeout |
| `GRANOLA_INFERENCE_API_KEY` | No | Inference API key |

Configuration is stored in `~/.config/granola-cli/config.json`.

## Google Setup

### Recommended: OAuth-First Setup

Use this when you want Gmail and Calendar access without gcloud quota-project friction.

1. Open Google Cloud Credentials:
   `https://console.cloud.google.com/apis/credentials`
2. Choose or create a Google Cloud project.
3. If prompted, configure the OAuth consent screen once.
4. Create a new OAuth client:
   - `Create Credentials` -> `OAuth client ID`
   - Application type: `Desktop app`
5. Copy the generated `Client ID` and `Client Secret`.
6. Run the CLI setup:

```bash
granola gmail setup-oauth
```

The CLI will prompt for the client ID and client secret once, save them to
`~/.config/granola-cli/config.json`, open the browser, and store the OAuth token
cache automatically.

Optional: save the credentials directly without the prompt:

```bash
granola config set google_client_id "<client-id>"
granola config set google_client_secret "<client-secret>"
granola gmail login
```

### ADC / gcloud Setup

Use this only if you explicitly want Application Default Credentials.

1. Sign in to gcloud:

```bash
gcloud auth login
```

2. Run one-shot setup:

```bash
granola gmail setup --project=<your-project-id> --login
```

That command will:
- enable `gmail.googleapis.com`
- enable `calendar-json.googleapis.com`
- set the ADC quota project
- run ADC login if `--login` is present

### Useful Google Cloud Links

- Credentials: `https://console.cloud.google.com/apis/credentials`
- OAuth consent screen: `https://console.cloud.google.com/apis/credentials/consent`
- Gmail API: `https://console.cloud.google.com/apis/library/gmail.googleapis.com`
- Google Calendar API: `https://console.cloud.google.com/apis/library/calendar-json.googleapis.com`

### Multi-Account Support

Granola tracks Google accounts in config and uses account-specific token caches.

```bash
granola gmail account list
granola gmail account use <email>
```

Tokens are not stored in plaintext config. Granola stores account metadata in
config and keeps OAuth tokens in local cache files.

## Usage

### Authentication

```bash
# Login with browser
granola auth login

# Logout
granola auth logout

# Check authentication status
granola auth status
```

### Meetings

```bash
# List meetings
granola meeting list --limit 20 --json

# View meeting details
granola meeting view <meeting-id>

# View meeting details with related attendee emails
granola meeting view <meeting-id> --email-context

# Get meeting transcript
granola meeting transcript <meeting-id>

# Get meeting notes
granola meeting notes <meeting-id>

# AI summary helpers
granola meeting <meeting-id> summarize
granola meeting <meeting-id> actions
granola meeting <meeting-id> key-takeaways
```

### Workspaces & Folders

```bash
# List workspaces
granola workspace list

# View workspace
granola workspace view <workspace-id>

# List folders
granola folder list

# View folder
granola folder view <folder-id>
```

### Gmail

```bash
# OAuth-first setup (recommended)
granola gmail setup-oauth

# Login to Google (once setup-oauth has saved the desktop client)
granola gmail login

# Optional: use gcloud ADC mode explicitly
granola gmail login --adc

# One-shot setup (enable APIs + set ADC quota project)
granola gmail setup

# Setup + login in one command
granola gmail setup --project=<your-project-id> --login

# Manage multiple Google accounts
granola gmail account list
granola gmail account use <email>

# List recent emails
granola gmail list

# Search emails
granola gmail list "meeting notes"

# Search emails involving a person (from/to/cc/bcc)
granola gmail person someone@example.com --max=10

# Advanced search with a person filter
granola gmail search "newer_than:30d" --person=someone@example.com --max=10

# Get email
granola gmail get <email-id>

# Get email with clean body extraction
granola gmail get <email-id> --body

# Get an entire thread
granola gmail thread <thread-id>

# Find emails from a meeting attendee
granola gmail from-attendee <email>

# Find emails around a meeting (uses attendees + time window)
granola gmail around-meeting <meeting-id>

# Persist related meeting/email links into the local DB
granola gmail link-meeting <meeting-id>
```

### Calendar

```bash
# OAuth-first setup also works from calendar command
granola calendar setup-oauth

# Login also works from calendar command
granola calendar login

# Setup also works from calendar command
granola calendar setup

# List upcoming events
granola calendar list

# Get event details
granola calendar get <event-id>

# Enrich meetings with calendar metadata
granola calendar enrich-meetings
```

### Embeddings & Semantic Search

Embeddings are not completely zero-setup. Before semantic search works well, you need:

1. Granola meeting data available locally
2. Ollama running locally
3. The `nomic-embed-text` model installed in Ollama

```bash
# Start Ollama if needed
ollama serve

# Install the embedding model once
ollama pull nomic-embed-text

# Make sure Granola meeting data is available locally
granola auth login
granola meeting list --limit 20
```

```bash
# Check embedding coverage
granola embedding status

# Generate embeddings for meetings missing vectors
granola embedding backfill

# Reset and rebuild all embeddings
granola embedding reset --force
granola embedding backfill

# Semantic search across meetings
granola search "meeting summary"

# Search as JSON for downstream tooling
granola search "project decisions" --json
```

Recommended semantic workflow:
1. `granola auth login`
2. `granola meeting list --limit 20`
3. `granola embedding status`
4. If coverage is not 100%, run `granola embedding backfill`
5. Use `granola search "..."` to find relevant meetings and transcript chunks

### Agent Workflows

These commands are useful for agents that need email, thread, meeting, and
calendar context together:

```bash
# Pull a clean body for downstream summarization or note linking
granola gmail get <email-id> --body --json

# Pull a whole thread for agent reasoning across message history
granola gmail thread <thread-id> --json

# Find emails involving a specific person, not just sent by them
granola gmail person someone@example.com --max=25 --json

# Find and store meeting-related email links in the local DB
granola gmail link-meeting <meeting-id>

# Show a meeting with related attendee email context
granola meeting view <meeting-id> --email-context

# Match meetings with calendar events for status/context enrichment
granola calendar enrich-meetings
```

### Context

```bash
# Attach context
granola context attach <meeting-id> "Important project decision"

# View progression
granola context progression <meeting-id>
```

### Configuration

```bash
# Get config path
granola config path

# List all config
granola config list

# Set config value
granola config set base_url "https://api.granola.ai/v1"

# Set model for AI helpers
granola config set model "mistral:latest"
```

## Troubleshooting

- If meeting data looks stale, run `granola auth login` again and re-list meetings.
- If semantic search returns weak or empty results, run `granola embedding status` and then `granola embedding backfill`.
- If Gmail or Calendar access fails, rerun `granola gmail login` or `granola gmail setup-oauth`.

## Acknowledgments

- Architecture inspiration from `https://github.com/magarcia/granola-cli` by @magarcia.
