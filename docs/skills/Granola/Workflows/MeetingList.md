# MeetingList

Use this workflow when the user wants a list of meetings.

## Primary Command

```bash
granola meeting list --limit 20 --json
```

## Variations

```bash
granola meeting list --limit 10
granola meeting list --limit 50 --json
```

## Notes

- Use JSON when the caller needs IDs or structured follow-up actions.
- If meeting data looks stale, run `granola auth login` and retry.
