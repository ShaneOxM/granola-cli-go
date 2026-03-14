---
name: Granola
description: Meeting notes and transcripts via Granola CLI. USE WHEN user asks about meetings, transcripts, notes, or meeting summaries.
allowed-tools: "Bash(granola:*)"
---

# Granola

Meeting notes, transcripts, Gmail context, and Calendar enrichment via Granola CLI (`granola` on PATH).

## Quick Actions

For most requests, use these patterns directly:

| User Says | Do This |
|-----------|---------|
| "list my meetings" | `Granola.MeetingList` |
| "what happened in meeting" | `Granola.MeetingView <id>` |
| "transcript for [meeting]" | `Granola.MeetingTranscript <id>` |
| "meeting notes" | `Granola.MeetingNotes <id>` |
| "summarize the meeting" | `granola meeting <id> summarize` |
| "show related emails" | `granola meeting view <id> --email-context` |
| "list my gmail" | `granola gmail list --max=10` |
| "list my calendar" | `granola calendar list --max=10` |
| "search emails involving someone" | `granola gmail person <email> --max=25` |
| "get the whole thread" | `granola gmail thread <thread-id> --json` |
| "search semantically" | `granola search "<query>"` |
| "check embeddings" | `granola embedding status` |

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **MeetingList** | "list meetings", "show meetings", "meetings this week" | `Workflows/MeetingList.md` |
| **MeetingView** | "view meeting", "meeting details", "what was in meeting" | `Workflows/MeetingView.md` |
| **MeetingTranscript** | "transcript", "transcript for", "full transcript" | `Workflows/MeetingTranscript.md` |
| **MeetingNotes** | "notes", "meeting notes", "summary" | `Workflows/MeetingNotes.md` |

## CLI Reference

### Google Setup

Preferred path is OAuth-first, similar to gws:

```bash
# One-time OAuth setup (recommended)
granola gmail setup-oauth

# Then log in through the browser callback flow
granola gmail login

# Optional account switching
granola gmail account list
granola gmail account use <email>
```

Google Cloud Console steps if client credentials are not already saved:
1. Credentials: `https://console.cloud.google.com/apis/credentials`
2. OAuth consent: `https://console.cloud.google.com/apis/credentials/consent`
3. Create an `OAuth client ID`
4. Choose `Desktop app`
5. Use the generated client ID + secret when prompted by `granola gmail setup-oauth`

Optional ADC path if explicitly needed:

```bash
granola gmail setup --project=<your-project-id> --login
granola gmail login --adc
```

### Authentication

**Check auth first when working with meetings. Gmail/Calendar have a separate OAuth-first login flow.**

```bash
# Check auth status
granola auth status 2>&1

# If authentication fails, try login
granola auth login

# If still fails, check if desktop app is running
ps aux | grep -i granola | grep -v grep

# If desktop app not running, start it
# granola desktop &  # Uncomment if needed

# If all fails, give up with clear error
echo "Error: Granola not authenticated. Please ensure Granola desktop app is running and try 'granola auth login'"
```

**Authentication Error Handling Flow:**
1. Try `granola auth status`
2. If failed: Try `granola auth login`
3. If failed: Check if desktop app is running
4. If not running: Inform user to start desktop app
5. If all fails: Give up with clear error message

**Note:** Granola is typically authenticated via desktop app integration. If auth fails, it usually means:
- Desktop app is not running
- Token has expired (run `granola auth login`)
- Network issue preventing sync

### Meetings

```bash
# List meetings
granola meeting list \
  --limit 20 \
  --json

# View meeting details
granola meeting view <id> --json

# View meeting details with email context from attendees
granola meeting view <id> --email-context

# Get transcript
granola meeting transcript <id> \
	--json

# Get notes
granola meeting notes <id>

# AI summary and agent helpers
granola meeting <id> summarize
granola meeting <id> actions
granola meeting <id> key-takeaways
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--limit <n>` | Number of meetings | 20 |
| `--json` | Output as JSON where supported | false |
| `--limit <n>` | Number of meetings | 20 |
| `--max=<n>` | Number of Gmail/Calendar results | 20 |

### Gmail & Calendar

```bash
# Gmail list/search
granola gmail list --max=10
granola gmail list "from:someone@example.com newer_than:7d" --max=10

# Search emails involving a person (from/to/cc/bcc)
granola gmail person someone@example.com --max=25
granola gmail search "newer_than:30d" --person=someone@example.com --max=25

# Gmail detail and meeting-aware search
granola gmail get <message-id>
granola gmail get <message-id> --body --json
granola gmail thread <thread-id> --json
granola gmail from-attendee <email>
granola gmail around-meeting <meeting-id>
granola gmail link-meeting <meeting-id>

# Calendar
granola calendar list --max=10
granola calendar get <event-id>
granola calendar enrich-meetings

# Semantic search + embeddings
granola embedding status
granola embedding backfill
granola search "meeting summary"
```

Embedding setup requirements:
1. Granola meeting data must already be available locally
2. Ollama must be running
3. `nomic-embed-text` must be installed

```bash
ollama serve
ollama pull nomic-embed-text
granola auth login
granola meeting list --limit 20
granola embedding backfill
```

## Examples

**Example 1: List recent meetings**
```bash
granola meeting list --limit 10 --json
```

**Example 2: Get transcript JSON**
```bash
granola meeting transcript abc123 --json
```

**Example 3: Semantic search**
```bash
granola search "Salesforce rollout discussion"
```

## Notes

- Granola meeting access is authenticated via desktop app integration
- Gmail/Calendar access is now OAuth-first and stored in Granola config + token cache
- Use `granola gmail get --body` and `granola gmail thread` when agents need clean bodies or full thread history
- Use `granola gmail link-meeting` plus `granola calendar enrich-meetings` to correlate notes, email, and calendar status in the local DB
- If semantic search is weak or empty, check `granola embedding status` and run `granola embedding backfill`
- For fresh data, trigger `~/.factory/scripts/granola-cli-sync.sh` before reading
- Meeting IDs are short alphanumeric strings (e.g., `abc123def456`)
