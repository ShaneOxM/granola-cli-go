# MeetingView

Use this workflow when the user wants details for a specific meeting.

## Primary Commands

```bash
granola meeting view <meeting-id> --json
granola meeting view <meeting-id> --email-context
```

## When To Use

- Meeting title, timestamps, attendees, workspace, and metadata
- Related email context for the meeting
- Follow-up investigation before transcript or notes retrieval

## Notes

- Prefer `--json` for agent follow-up work.
- Use `--email-context` when user wants related Gmail context from attendees.
