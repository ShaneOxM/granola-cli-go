# MeetingNotes

Use this workflow when the user wants notes or summary-oriented meeting output.

## Primary Commands

```bash
granola meeting notes <meeting-id>
granola meeting <meeting-id> summarize
granola meeting <meeting-id> actions
granola meeting <meeting-id> key-takeaways
```

## Notes

- `granola meeting notes` returns stored notes content when present.
- If notes are empty, prefer transcript + AI helper commands.
- For semantic recall across meetings, use `granola search "<query>"` after embeddings are ready.
