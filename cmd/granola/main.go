package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ShaneOxM/granola-cli-go/internal/api"
	"github.com/ShaneOxM/granola-cli-go/internal/auth"
	cal "github.com/ShaneOxM/granola-cli-go/internal/calendar"
	"github.com/ShaneOxM/granola-cli-go/internal/config"
	"github.com/ShaneOxM/granola-cli-go/internal/embeddings"
	gm "github.com/ShaneOxM/granola-cli-go/internal/gmail"
	"github.com/ShaneOxM/granola-cli-go/internal/inference"
	"github.com/ShaneOxM/granola-cli-go/internal/logger"
	meet "github.com/ShaneOxM/granola-cli-go/internal/meeting"
	"github.com/ShaneOxM/granola-cli-go/internal/output"
	"github.com/ShaneOxM/granola-cli-go/internal/storage"
	"github.com/ShaneOxM/granola-cli-go/internal/utils"
)

var granolaClient *api.Client
var db *storage.DB

const (
	defaultBatchSize = 10
	defaultLimit     = 100
)

func main() {
	if err := config.Init(); err != nil {
		logger.Error("Error initializing config", "error", err)
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	if err := auth.Init(); err != nil {
		logger.Debug("Auth initialization deferred", "error", err)
	}

	granolaClient = api.NewClient()

	configPath, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get config path: %v\n", err)
	}
	dbPath := configPath
	if dbPath != "" {
		dbPath = dbPath[:len(dbPath)-len("config.json")] + "granola.db"
	}
	db, err = storage.NewDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to open database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Some features may be unavailable\n")
		db = nil
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "auth":
		runAuth(os.Args[2:])
	case "meeting":
		runMeeting(os.Args[2:])
	case "workspace":
		runWorkspace(os.Args[2:])
	case "folder":
		runFolder(os.Args[2:])
	case "gmail":
		runGmail(os.Args[2:])
	case "calendar":
		runCalendar(os.Args[2:])
	case "context":
		runContext(os.Args[2:])

	case "config":
		runConfig(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "embedding":
		runEmbedding(os.Args[2:])
	case "--version", "-v":
		fmt.Println("0.4.0-go")
	case "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Granola CLI - Meeting notes CLI")
	fmt.Println()
	fmt.Println("Usage: granola <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  auth        Authentication (login, logout, status)")
	fmt.Println("  meeting     Meetings (list, view, transcript, notes)")
	fmt.Println("  workspace   Workspaces (list, view)")
	fmt.Println("  folder      Folders (list, view)")
	fmt.Println("  gmail       Gmail (OAuth-first: setup-oauth, login, list, get, from-attendee, around-meeting)")
	fmt.Println("  calendar    Calendar (list, get, enrich-meetings)")
	fmt.Println("  context     Context (attach, progression)")
	fmt.Println("  config      Configuration (profile, list, set, use)")
	fmt.Println("  search      Semantic search across meetings")
	fmt.Println("  embedding   Embedding management (status, backfill, reset)")
	fmt.Println("  agent       Agent tools (coming soon)")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -h, --help      Show help")
	fmt.Println("  -v, --version   Show version")
	fmt.Println("  -j, --json      JSON output")
}

func runAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("Auth commands: login, logout, status")
		return
	}
	switch args[0] {
	case "login":
		runLogin(args[1:])
	case "logout":
		runLogout(args[1:])
	case "status":
		if err := auth.Status(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown auth command: %s\n", args[0])
	}
}

func runLogin(args []string) {
	if err := auth.Login(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runLogout(args []string) {
	if err := auth.Logout(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMeeting(args []string) {
	if len(args) < 1 {
		fmt.Println("Meeting commands: list, view, transcript, notes, summarize, actions, key-takeaways, sentiment-analysis, discussion-questions")
		return
	}

	// Check if first arg is a meeting ID (contains hyphens, looks like UUID)
	var meetingID string
	var subArgs []string

	// Check if first argument looks like a UUID (contains hyphens)
	if len(args) > 0 && strings.Contains(args[0], "-") {
		// Check if second arg is a command
		if len(args) > 1 && isAICommand(args[1]) {
			meetingID = args[0]
			subArgs = args[1:]
		}
	}

	if meetingID != "" {
		// Pass meeting ID to AI commands
		switch subArgs[0] {
		case "summarize":
			runAISummarize(append([]string{meetingID}, subArgs[1:]...))
		case "actions":
			runAIActions(append([]string{meetingID}, subArgs[1:]...))
		case "key-takeaways":
			runAIKeyTakeaways(append([]string{meetingID}, subArgs[1:]...))
		case "sentiment-analysis":
			runAISentimentAnalysis(append([]string{meetingID}, subArgs[1:]...))
		case "discussion-questions":
			runAIDiscussionQuestions(append([]string{meetingID}, subArgs[1:]...))
		default:
			fmt.Fprintf(os.Stderr, "Unknown meeting command: %s\n", subArgs[0])
		}
		return
	}

	switch args[0] {
	case "list":
		runMeetingList(args[1:])
	case "view":
		runMeetingView(args[1:])
	case "transcript":
		runMeetingTranscript(args[1:])
	case "notes":
		runMeetingNotes(args[1:])
	case "summarize":
		runAISummarize(args[1:])
	case "actions":
		runAIActions(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown meeting command: %s\n", args[0])
	}
}

func runMeetingList(args []string) {
	limit := 20
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				if l, err := strconv.Atoi(args[i+1]); err == nil {
					limit = l
				}
				i++
			}
		case "--json":
			jsonOutput = true
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching from Granola CLI...\n")
	meetings, err := granolaClient.ListMeetings(limit, "", "", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching from Granola: %v\n", err)
		os.Exit(1)
	}

	if db != nil {
		for _, m := range meetings {
			dbMeeting := &storage.Meeting{
				ID:          m.ID,
				Title:       m.Title,
				WorkspaceID: m.WorkspaceID,
				CreatedAt:   m.CreatedAt,
				UpdatedAt:   m.UpdatedAt,
			}
			db.SaveMeeting(context.Background(), dbMeeting)
		}
	}

	if jsonOutput {
		output.JSON(meetings)
	} else {
		fmt.Printf("Found %d meetings\n\n", len(meetings))
		for _, m := range meetings {
			fmt.Printf("%s - %s\n", m.ID, m.Title)
		}
	}
}

func runMeetingView(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: granola meeting view <id> [--json] [--email-context]")
		return
	}
	id := args[0]
	jsonOutput := false
	emailContext := false
	for _, arg := range args[1:] {
		if arg == "--json" {
			jsonOutput = true
		}
		if arg == "--email-context" {
			emailContext = true
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching from Granola CLI...\n")
	m, err := granolaClient.GetMeeting(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		output.JSON(m)
	} else {
		fmt.Printf("Meeting: %s\n", m.Title)
		fmt.Printf("ID: %s\n", m.ID)
		fmt.Printf("Created: %s\n", m.CreatedAt)
		fmt.Printf("Updated: %s\n", m.UpdatedAt)
		if m.WorkspaceID != "" {
			fmt.Printf("Workspace: %s\n", m.WorkspaceID)
		}
		if len(m.Attendees) > 0 {
			fmt.Println("\nAttendees:")
			for _, a := range m.Attendees {
				if a.Email != "" {
					if a.Name != "" {
						fmt.Printf("  - %s (%s)\n", a.Name, a.Email)
					} else {
						fmt.Printf("  - %s\n", a.Email)
					}
				}
			}
		}

		if emailContext {
			ga, aerr := auth.NewGmailCalendarAuth()
			if aerr != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: Gmail auth unavailable: %v\n", aerr)
				return
			}
			gc, gerr := gm.NewClient(context.Background(), ga)
			if gerr != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: Gmail client unavailable: %v\n", gerr)
				return
			}
			emails, eerr := meet.AroundMeetingEmails(context.Background(), gc, *m, 20)
			if eerr != nil {
				fmt.Fprintf(os.Stderr, "\nWarning: Failed to load email context: %v\n", eerr)
				return
			}
			if len(emails) == 0 {
				fmt.Println("\nEmail Context: none found")
			} else {
				fmt.Println("\nEmail Context:")
				for _, em := range emails {
					fmt.Printf("  - %s | %s | %s\n", em.Date, em.From, em.Subject)
				}
			}
		}

		if db != nil {
			if links, err := db.GetEmailLinks(m.ID); err == nil && len(links) > 0 {
				fmt.Println("\nStored Email Links:")
				for _, l := range links {
					fmt.Printf("  - %s (%s, score %d)\n", l.EmailID, l.Reason, l.Score)
				}
			}
			if links, err := db.GetCalendarLinks(m.ID); err == nil && len(links) > 0 {
				fmt.Println("\nStored Calendar Links:")
				for _, l := range links {
					fmt.Printf("  - %s (%s, score %d)\n", l.EventSummary, l.Reason, l.Score)
				}
			}
		}
	}
}

func runMeetingTranscript(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: granola meeting transcript <id>")
		return
	}
	id := args[0]
	jsonOutput := false
	clipboardCopy := false
	outputFile := ""
	outputFormat := "md"

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--clipboard":
			clipboardCopy = true
		case "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				outputFormat = args[i+1]
				i++
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching transcript for: %s\n", id)
	transcript, err := granolaClient.GetTranscript(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(transcript) == 0 {
		fmt.Println("No transcript found")
		return
	}

	if jsonOutput {
		output.JSON(transcript)
		return
	}

	var transcriptText string
	if outputFormat == "json" {
		jsonData, _ := json.MarshalIndent(transcript, "", "  ")
		transcriptText = string(jsonData)
	} else {
		transcriptText = formatTranscriptMarkdown(transcript)
	}

	if outputFile != "" {
		if err := validateAndWriteFile(outputFile, []byte(transcriptText), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Saved to %s\n", outputFile)
	}

	if clipboardCopy {
		if err := utils.CopyToClipboard(transcriptText); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not copy to clipboard: %v\n", err)
		} else {
			fmt.Println("✓ Copied to clipboard")
		}
	}

	fmt.Println(transcriptText)
}

func runMeetingNotes(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: granola meeting notes <id>")
		return
	}
	id := args[0]
	jsonOutput := false
	for _, arg := range args[1:] {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching notes for: %s\n", id)
	doc, err := granolaClient.GetNotes(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if doc == nil {
		fmt.Println("No notes found")
		return
	}

	if jsonOutput {
		output.JSON(doc)
	} else {
		text := renderProseMirrorText(doc)
		if strings.TrimSpace(text) == "" {
			fmt.Println("No notes content yet")
			return
		}
		fmt.Println(text)
	}
}

func runWorkspace(args []string) {
	if len(args) < 1 {
		fmt.Println("Workspace commands: list, view")
		return
	}
	switch args[0] {
	case "list":
		jsonOutput := false
		for _, arg := range args[1:] {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		fmt.Fprintf(os.Stderr, "Fetching workspaces from Granola CLI...\n")
		workspaces, err := granolaClient.ListWorkspaces()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			output.JSON(workspaces)
			return
		}
		fmt.Printf("\nFound %d workspaces\n\n", len(workspaces))
		for _, w := range workspaces {
			fmt.Printf("%s - %s\n", w.ID, w.Name)
		}
	case "view":
		if len(args) < 2 {
			fmt.Println("Usage: granola workspace view <id>")
			return
		}
		jsonOutput := false
		for _, arg := range args[2:] {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		workspace, err := granolaClient.GetWorkspace(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			output.JSON(workspace)
			return
		}
		fmt.Printf("Workspace: %s\nID: %s\nSlug: %s\nCreated: %s\nUpdated: %s\n", workspace.Name, workspace.ID, workspace.Slug, workspace.CreatedAt, workspace.UpdatedAt)
	default:
		fmt.Fprintf(os.Stderr, "Unknown workspace command: %s\n", args[0])
	}
}

func runFolder(args []string) {
	if len(args) < 1 {
		fmt.Println("Folder commands: list, view")
		return
	}
	switch args[0] {
	case "list":
		jsonOutput := false
		for _, arg := range args[1:] {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		fmt.Fprintf(os.Stderr, "Fetching folders from Granola CLI...\n")
		folders, err := granolaClient.ListFolders("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			output.JSON(folders)
			return
		}
		fmt.Printf("\nFound %d folders\n\n", len(folders))
		for _, f := range folders {
			name := f.Name
			if name == "" {
				name = f.Name
			}
			if name == "" {
				name = f.ID
			}
			fmt.Printf("%s - %s\n", f.ID, name)
		}
	case "view":
		if len(args) < 2 {
			fmt.Println("Usage: granola folder view <id>")
			return
		}
		jsonOutput := false
		for _, arg := range args[2:] {
			if arg == "--json" {
				jsonOutput = true
			}
		}
		folder, err := granolaClient.GetFolder(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if jsonOutput {
			output.JSON(folder)
			return
		}
		fmt.Printf("Folder: %s\nID: %s\nSlug: %s\nWorkspace: %s\nCreated: %s\nUpdated: %s\n", folder.Name, folder.ID, folder.Slug, folder.WorkspaceID, folder.CreatedAt, folder.UpdatedAt)
	default:
		fmt.Fprintf(os.Stderr, "Unknown folder command: %s\n", args[0])
	}
}

func runGmail(args []string) {
	if len(args) < 1 {
		fmt.Println("Gmail commands: setup, setup-oauth, login, account, list, search, get, thread, person, from-attendee, around-meeting, link-meeting")
		return
	}
	if args[0] == "setup-oauth" {
		runGoogleSetupOAuth()
		return
	}
	if args[0] == "setup" {
		runGoogleSetup(args[1:])
		return
	}
	if args[0] == "login" {
		runGoogleLogin(args[1:])
		return
	}
	if args[0] == "account" {
		runGoogleAccount(args[1:])
		return
	}
	ga, err := auth.NewGmailCalendarAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	gc, err := gm.NewClient(context.Background(), ga)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		fallthrough
	case "search":
		opts := parseGmailListOptions(args[1:])
		var msgs []gm.Message
		if opts.person != "" {
			msgs, err = gc.ListInvolvingPerson(context.Background(), opts.person, opts.max)
		} else {
			msgs, err = gc.ListMessages(context.Background(), opts.query, opts.max)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(msgs) == 0 {
			fmt.Println("No messages found")
			return
		}
		if opts.jsonOutput {
			output.JSON(msgs)
			return
		}
		_ = output.Table([]string{"Date", "From", "Subject"}, gmailRows(msgs))
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail get <id>")
			return
		}
		opts := parseGmailGetOptions(args[2:])
		msg, err := gc.GetMessage(context.Background(), args[1], opts.format)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if opts.jsonOutput {
			output.JSON(msg)
			return
		}
		fmt.Printf("ID: %s\nThread: %s\nDate: %s\nFrom: %s\nTo: %s\nCc: %s\nSubject: %s\n\n", msg.ID, msg.ThreadID, msg.Date, msg.From, msg.To, msg.Cc, msg.Subject)
		if opts.bodyOutput && msg.Body != "" {
			fmt.Printf("Body:\n%s\n", msg.Body)
		} else {
			fmt.Printf("Snippet:\n%s\n", msg.Snippet)
		}
	case "thread":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail thread <thread-id> [--json]")
			return
		}
		jsonOutput := false
		for _, a := range args[2:] {
			if a == "--json" {
				jsonOutput = true
			}
		}
		thread, err := gc.GetThread(context.Background(), args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if jsonOutput {
			output.JSON(thread)
			return
		}
		_ = output.Table([]string{"Date", "From", "Subject"}, gmailRows(thread.Messages))
	case "person":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail person <email-or-name> [--max=N]")
			return
		}
		max := int64(20)
		for _, a := range args[2:] {
			if strings.HasPrefix(a, "--max=") {
				if n, err := strconv.ParseInt(strings.TrimPrefix(a, "--max="), 10, 64); err == nil {
					max = n
				}
			}
		}
		msgs, err := gc.ListInvolvingPerson(context.Background(), args[1], max)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(msgs) == 0 {
			fmt.Println("No messages found")
			return
		}
		_ = output.Table([]string{"Date", "From", "Subject"}, gmailRows(msgs))
	case "from-attendee":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail from-attendee <email>")
			return
		}
		msgs, err := gc.ListFromAttendee(context.Background(), args[1], 20)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(msgs) == 0 {
			fmt.Println("No messages found")
			return
		}
		_ = output.Table([]string{"Date", "Subject"}, gmailSubjectRows(msgs))
	case "around-meeting":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail around-meeting <meeting-id>")
			return
		}
		m, err := granolaClient.GetMeeting(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading meeting: %v\n", err)
			os.Exit(1)
		}
		msgs, err := meet.AroundMeetingEmails(context.Background(), gc, *m, 30)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(msgs) == 0 {
			fmt.Println("No messages found around this meeting")
			return
		}
		_ = output.Table([]string{"Date", "From", "Subject"}, gmailRows(msgs))
	case "link-meeting":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail link-meeting <meeting-id>")
			return
		}
		m, err := granolaClient.GetMeeting(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading meeting: %v\n", err)
			os.Exit(1)
		}
		msgs, err := meet.LinkMeetingEmails(context.Background(), db, gc, *m, 30)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(msgs) == 0 {
			fmt.Println("No related emails found to link")
			return
		}
		rows := make([][]string, 0, len(msgs))
		for _, msg := range msgs {
			rows = append(rows, []string{output.Truncate(msg.ID, 22), output.Truncate(msg.ThreadID, 22), output.Truncate(msg.Subject, 60)})
		}
		_ = output.Table([]string{"Email ID", "Thread ID", "Subject"}, rows)
	default:
		fmt.Fprintf(os.Stderr, "Unknown gmail command: %s\n", args[0])
	}
}

func runCalendar(args []string) {
	if len(args) < 1 {
		fmt.Println("Calendar commands: setup, setup-oauth, login, account, list, get, enrich-meetings")
		return
	}
	if args[0] == "setup-oauth" {
		runGoogleSetupOAuth()
		return
	}
	if args[0] == "setup" {
		runGoogleSetup(args[1:])
		return
	}
	if args[0] == "login" {
		runGoogleLogin(args[1:])
		return
	}
	if args[0] == "account" {
		runGoogleAccount(args[1:])
		return
	}
	ga, err := auth.NewGmailCalendarAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cc, err := cal.NewClient(context.Background(), ga)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		opts := parseCalendarListOptions(args[1:])
		events, err := cc.ListEvents(context.Background(), opts.calendarID, time.Time{}, time.Time{}, opts.max)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(events) == 0 {
			fmt.Println("No events found")
			return
		}
		if opts.jsonOutput {
			output.JSON(events)
			return
		}
		_ = output.Table([]string{"Start", "Title", "Event ID"}, calendarRows(events))
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: granola calendar get <event-id>")
			return
		}
		e, err := cc.GetEvent(context.Background(), "primary", args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		fmt.Printf("Title: %s\nStart: %s\nEnd: %s\nLocation: %s\n\nDescription:\n%s\n", e.Summary, e.Start, e.End, e.Location, e.Description)
	case "enrich-meetings":
		meetings, err := granolaClient.ListMeetings(50, "", "", "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		enriched, err := meet.EnrichMeetings(context.Background(), cc, meetings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printGoogleQuotaHint(err)
			os.Exit(1)
		}
		if len(enriched) == 0 {
			fmt.Println("No meetings found to enrich")
			return
		}
		if db != nil {
			for _, item := range enriched {
				if item.MatchedEvent != nil {
					_ = db.SaveCalendarLink(item.Meeting.ID, item.MatchedEvent.ID, item.MatchedEvent.Summary, item.MatchedBy, item.MatchScore)
				}
			}
		}
		_ = output.Table([]string{"Meeting", "Calendar Match", "Score", "Status"}, enrichedMeetingRows(enriched))
	default:
		fmt.Fprintf(os.Stderr, "Unknown calendar command: %s\n", args[0])
	}
}

func runGoogleLogin(args []string) {
	useADC := false
	for _, a := range args {
		if a == "--adc" {
			useADC = true
		}
	}

	ga, err := auth.NewGmailCalendarAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !ga.CanInteractiveLogin() {
		if !useADC {
			fmt.Println("No OAuth client configured. Enter Google OAuth client credentials once (saved to granola config).")
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Client ID: ")
			cid, _ := reader.ReadString('\n')
			fmt.Print("Client Secret: ")
			csec, _ := reader.ReadString('\n')
			cid = strings.TrimSpace(cid)
			csec = strings.TrimSpace(csec)
			if cid == "" || csec == "" {
				fmt.Fprintln(os.Stderr, "Missing client credentials. You can also use ADC with: granola gmail login --adc")
				os.Exit(1)
			}
			if err := config.Set("google_client_id", cid); err != nil {
				fmt.Fprintf(os.Stderr, "Failed saving google_client_id: %v\n", err)
				os.Exit(1)
			}
			if err := config.Set("google_client_secret", csec); err != nil {
				fmt.Fprintf(os.Stderr, "Failed saving google_client_secret: %v\n", err)
				os.Exit(1)
			}
			ga, err = auth.NewGmailCalendarAuth()
			if err != nil || !ga.CanInteractiveLogin() {
				fmt.Fprintf(os.Stderr, "OAuth client setup failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Using gcloud ADC login...")
			cmd := exec.Command("gcloud", "auth", "application-default", "login", "--scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/gmail.readonly,https://www.googleapis.com/auth/calendar.readonly")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "gcloud login failed: %v\n", err)
				os.Exit(1)
			}
			if acctOut, err := exec.Command("gcloud", "config", "get-value", "account").Output(); err == nil {
				email := strings.TrimSpace(string(acctOut))
				if email != "" && email != "(unset)" {
					_ = ga.ActivateAccount(email)
				}
				_ = saveGoogleAuthState(email, "adc")
			}
			fmt.Println("✓ Google login successful via gcloud ADC")
			return
		}
	}

	state := fmt.Sprintf("granola-%d", time.Now().Unix())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start callback listener: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	ga.SetRedirectURL(fmt.Sprintf("http://localhost:%d", port))
	url, err := ga.AuthCodeURL(state)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth state mismatch"):
			default:
			}
			return
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("missing authorization code"):
			default:
			}
			return
		}
		_, _ = w.Write([]byte("Granola login successful. You can close this tab."))
		select {
		case codeCh <- code:
		default:
		}
	})
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	fmt.Println("Opening browser for Google login...")
	if openErr := exec.Command("open", url).Run(); openErr != nil {
		fmt.Println("Open this URL in your browser and approve access:")
		fmt.Println(url)
	}

	var code string
	select {
	case code = <-codeCh:
	case waitErr := <-errCh:
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", waitErr)
		os.Exit(1)
	case <-time.After(5 * time.Minute):
		fmt.Fprintln(os.Stderr, "Login timed out waiting for browser callback")
		os.Exit(1)
	}
	_ = server.Shutdown(context.Background())

	if err := ga.ExchangeCode(context.Background(), code); err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}
	_ = saveGoogleAuthMode("oauth")
	if email, err := ga.CurrentEmail(context.Background()); err == nil {
		_ = ga.ActivateAccount(email)
		_ = saveGoogleAuthState(email, "oauth")
	}

	fmt.Println("✓ Google Gmail/Calendar login successful")
}

func runGoogleSetupOAuth() {
	fmt.Println("OAuth-focused setup")
	fmt.Println("- Stores Google OAuth client credentials in granola config")
	fmt.Println("- Uses browser OAuth login and local token cache")
	fmt.Println("- Avoids ADC project/quota setup unless you explicitly choose --adc")
	runGoogleLogin(nil)
}

func runGoogleSetup(args []string) {
	project := ""
	autoLogin := false
	for _, a := range args {
		if strings.HasPrefix(a, "--project=") {
			project = strings.TrimSpace(strings.TrimPrefix(a, "--project="))
		}
		if a == "--login" {
			autoLogin = true
		}
	}
	if project == "" {
		projectOut, err := exec.Command("gcloud", "config", "get-value", "project").Output()
		if err == nil {
			project = strings.TrimSpace(string(projectOut))
		}
	}

	if project == "" || project == "(unset)" {
		if picked, err := pickFirstGcloudProject(); err == nil && picked != "" {
			project = picked
			fmt.Printf("No active project found. Auto-selected project: %s\n", project)
		} else {
			fmt.Fprintln(os.Stderr, "No active gcloud project found and none auto-discovered.")
			fmt.Fprintln(os.Stderr, "Continue with login only: granola gmail login")
			fmt.Fprintln(os.Stderr, "Or set one with: gcloud config set project <project-id> or pass --project=<project-id>")
			if autoLogin {
				runGoogleLogin(nil)
			}
			return
		}
	}

	if err := exec.Command("gcloud", "config", "set", "project", project).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not set active project globally: %v\n", err)
	}

	fmt.Printf("Using project: %s\n", project)
	enable := exec.Command("gcloud", "services", "enable", "gmail.googleapis.com", "calendar-json.googleapis.com", "--project", project)
	enable.Stdout = os.Stdout
	enable.Stderr = os.Stderr
	if err := enable.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed enabling APIs: %v\n", err)
		fmt.Fprintln(os.Stderr, "You can still continue if APIs are already enabled on your project.")
	}

	setQuota := exec.Command("gcloud", "auth", "application-default", "set-quota-project", project)
	setQuota.Stdout = os.Stdout
	setQuota.Stderr = os.Stderr
	if err := setQuota.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed setting quota project: %v\n", err)
		fmt.Fprintln(os.Stderr, "Ensure your account has serviceusage.services.use on this project.")
		fmt.Fprintln(os.Stderr, "You can still try commands; if quota errors appear, use a project where you have permission.")
	}

	fmt.Println("✓ Google setup complete (APIs enabled + quota project set)")
	if autoLogin {
		runGoogleLogin(nil)
	}
	return
}

func pickFirstGcloudProject() (string, error) {
	cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId)", "--limit=1")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGoogleAccount(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: granola gmail account <list|use>")
		return
	}
	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.GoogleAccounts == nil {
		cfg.GoogleAccounts = map[string]string{}
	}

	switch args[0] {
	case "list":
		if len(cfg.GoogleAccounts) == 0 {
			fmt.Println("No Google accounts configured yet. Run: granola gmail login")
			return
		}
		fmt.Println("Google accounts:")
		for email, mode := range cfg.GoogleAccounts {
			active := ""
			if email == cfg.GoogleActiveAccount {
				active = " (active)"
			}
			fmt.Printf("  - %s [%s]%s\n", email, mode, active)
		}
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: granola gmail account use <email>")
			return
		}
		email := strings.TrimSpace(args[1])
		if _, ok := cfg.GoogleAccounts[email]; !ok {
			fmt.Fprintf(os.Stderr, "Unknown account: %s\n", email)
			os.Exit(1)
		}
		cfg.GoogleActiveAccount = email
		if err := config.Write(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Active Google account set to %s\n", email)
	default:
		fmt.Fprintf(os.Stderr, "Unknown account command: %s\n", args[0])
	}
}

func saveGoogleAuthState(email, mode string) error {
	email = strings.TrimSpace(email)
	cfg, err := config.Read()
	if err != nil {
		return err
	}
	cfg.GoogleAuthMode = mode
	if email == "" || email == "(unset)" {
		return config.Write(cfg)
	}
	if cfg.GoogleAccounts == nil {
		cfg.GoogleAccounts = map[string]string{}
	}
	cfg.GoogleAccounts[email] = mode
	cfg.GoogleActiveAccount = email
	return config.Write(cfg)
}

func saveGoogleAuthMode(mode string) error {
	cfg, err := config.Read()
	if err != nil {
		return err
	}
	cfg.GoogleAuthMode = mode
	return config.Write(cfg)
}

func printGoogleQuotaHint(err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "quota project") || strings.Contains(msg, "accessNotConfigured") || strings.Contains(msg, "SERVICE_DISABLED") {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Google API setup hint:")
		fmt.Fprintln(os.Stderr, "1) Choose a valid GCP project you can bill/use")
		fmt.Fprintln(os.Stderr, "2) Enable Gmail API and Google Calendar API")
		fmt.Fprintln(os.Stderr, "3) Set ADC quota project:")
		fmt.Fprintln(os.Stderr, "   gcloud auth application-default set-quota-project <your-project-id>")
	}
}

func runContext(args []string) {
	if len(args) < 1 {
		fmt.Println("Context commands: attach, progression")
		return
	}
	switch args[0] {
	case "attach":
		if len(args) < 3 {
			fmt.Println("Usage: granola context attach <meeting-id> <context>")
			return
		}
		meetingID := args[1]
		context := args[2]
		if db != nil {
			if err := db.AttachContext(meetingID, "context", context); err != nil {
				fmt.Fprintf(os.Stderr, "Error attaching context: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Printf("✓ Attached context to meeting %s\n", meetingID)
	case "progression":
		if len(args) < 2 {
			fmt.Println("Usage: granola context progression <meeting-id>")
			return
		}
		meetingID := args[1]
		if db != nil {
			progressions, err := db.GetProgression(meetingID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if len(progressions) == 0 {
				fmt.Printf("No progression recorded for meeting %s\n", meetingID)
				return
			}
			fmt.Printf("\nProgression for %s:\n\n", meetingID)
			fmt.Printf("%-25s %-20s %s\n", "Date", "Stage", "Description")
			fmt.Println("----------------------------------------------------------------------")
			for _, p := range progressions {
				date := p.CreatedAt.Format(time.RFC3339)
				fmt.Printf("%-25s %-20s %s\n", date, p.Stage, p.Description)
			}
		} else {
			fmt.Println("Database not available")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown context command: %s\n", args[0])
	}
}

func runConfig(args []string) {
	if len(args) < 1 {
		fmt.Println("Config commands: list, set, use, profile")
		return
	}
	switch args[0] {
	case "path":
		path, err := config.Path()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Println(path)
	case "list":
		cfg, err := config.Read()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config: %+v\n", cfg)
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: granola config set <key> <value>")
			fmt.Println("\nAvailable keys:")
			fmt.Println("  base_url    Inference endpoint URL")
			fmt.Println("  model       Default model name")
			fmt.Println("  timeout     Timeout in seconds (default: 300)")
			fmt.Println("  google_client_id      Google OAuth client ID")
			fmt.Println("  google_client_secret  Google OAuth client secret")
			return
		}
		if err := config.Set(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Set %s=%s\n", args[1], args[2])
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: granola config use <profile>")
			fmt.Println("\nAvailable profiles:")
			profiles := config.ListProfiles()
			for _, p := range profiles {
				fmt.Printf("  - %s\n", p)
			}
			fmt.Printf("\nDefault profile: %s\n", config.GetDefaultProfile())
			return
		}
		if err := config.UseProfile(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Using profile: %s\n", args[1])
	case "profile":
		if len(args) < 2 {
			fmt.Println("Profile commands: list")
			return
		}
		switch args[1] {
		case "list":
			profiles := config.ListProfiles()
			fmt.Println("Available profiles:")
			for _, p := range profiles {
				fmt.Printf("  - %s\n", p)
			}
			fmt.Printf("\nDefault profile: %s\n", config.GetDefaultProfile())
		default:
			fmt.Fprintf(os.Stderr, "Unknown profile command: %s\n", args[1])
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", args[0])
	}
}

func formatTranscriptMarkdown(transcript []api.Utterance) string {
	var sb strings.Builder
	sb.WriteString("## Meeting Transcript\n\n")

	for _, u := range transcript {
		source := u.Source
		if source == "microphone" {
			source = "Microphone"
		} else if source == "system" {
			source = "System"
		}
		sb.WriteString(fmt.Sprintf("**[%s]** %s: %s\n\n",
			u.StartTimestamp,
			source,
			u.Text))
	}

	return sb.String()
}

// isAICommand checks if a string is an AI command
func isAICommand(cmd string) bool {
	aiCommands := []string{"summarize", "actions", "key-takeaways", "sentiment-analysis", "discussion-questions"}
	for _, c := range aiCommands {
		if cmd == c {
			return true
		}
	}
	return false
}

func runAISummarize(args []string) {
	executeAICommand("summarize", args, func(client *inference.Client, text string, limit int) (string, error) {
		return client.Summarize(text, limit)
	})
}

func runAIActions(args []string) {
	executeAICommand("actions", args, func(client *inference.Client, text string, limit int) (string, error) {
		return client.ExtractActions(text, limit)
	})
}

func runAIKeyTakeaways(args []string) {
	executeAICommand("key-takeaways", args, func(client *inference.Client, text string, limit int) (string, error) {
		return client.KeyTakeaways(text, limit)
	})
}

func runAISentimentAnalysis(args []string) {
	executeAICommand("sentiment-analysis", args, func(client *inference.Client, text string, limit int) (string, error) {
		return client.SentimentAnalysis(text, limit)
	})
}

func runAIDiscussionQuestions(args []string) {
	executeAICommand("discussion-questions", args, func(client *inference.Client, text string, limit int) (string, error) {
		return client.GenerateQuestions(text, limit)
	})
}

func executeAICommand(cmdName string, args []string, aiFunc func(*inference.Client, string, int) (string, error)) {
	if len(args) < 1 {
		fmt.Printf("Usage: granola meeting <id> %s [--clipboard]\n", cmdName)
		return
	}

	prefixes := map[string]string{
		"summarize":            "## Summary",
		"actions":              "## Action Items",
		"key-takeaways":        "## Key Takeaways",
		"sentiment-analysis":   "## Sentiment Analysis",
		"discussion-questions": "## Discussion Questions",
	}

	descs := map[string]string{
		"summarize":            "Summarizing",
		"actions":              "Extracting actions",
		"key-takeaways":        "Extracting key takeaways",
		"sentiment-analysis":   "Analyzing sentiment",
		"discussion-questions": "Generating discussion questions",
	}

	meetingID := args[0]
	clipboardCopy := false

	for _, arg := range args[1:] {
		if arg == "--clipboard" {
			clipboardCopy = true
		}
	}

	fmt.Fprintf(os.Stderr, "Fetching transcript for: %s\n", meetingID)
	transcript, err := granolaClient.GetTranscript(meetingID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var transcriptText strings.Builder
	for _, u := range transcript {
		transcriptText.WriteString("[")
		transcriptText.WriteString(u.StartTimestamp)
		transcriptText.WriteString("] ")
		transcriptText.WriteString(u.Source)
		transcriptText.WriteString(": ")
		transcriptText.WriteString(u.Text)
		transcriptText.WriteString("\n")
	}

	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.BaseURL == "" || cfg.Model == "" {
		fmt.Fprintf(os.Stderr, "Error: Inference endpoint or model not configured\n")
		fmt.Fprintf(os.Stderr, "Run: granola config set base_url <url>\n")
		fmt.Fprintf(os.Stderr, "Run: granola config set model <model-name>\n")
		fmt.Fprintf(os.Stderr, "See README sections: Configuration and Embeddings & Semantic Search.\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "%s with %s...\n", descs[cmdName], cfg.Model)
	inferenceClient := inference.NewClient(cfg.BaseURL, cfg.Model)

	result, err := aiFunc(inferenceClient, transcriptText.String(), 1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		printInferenceSetupHint(err, cfg)
		os.Exit(1)
	}

	fmt.Printf("\n%s\n\n", prefixes[cmdName])
	fmt.Println(result)

	if clipboardCopy {
		if err := utils.CopyToClipboard(prefixes[cmdName] + "\n\n" + result); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not copy to clipboard: %v\n", err)
		} else {
			fmt.Println("\n✓ Copied to clipboard")
		}
	}
}

func runSearch(args []string) {
	if db == nil {
		fmt.Fprintf(os.Stderr, "Error: Database not available. Please check your configuration.\n")
		os.Exit(1)
	}
	if len(args) < 1 {
		fmt.Println("Usage: granola search <query> [flags]")
		fmt.Println("       granola meeting <id> search <query> [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --since DATE       Filter by date (YYYY-MM-DD)")
		fmt.Println("  --workspace ID     Filter by workspace")
		fmt.Println("  --top N            Number of results (default: 10)")
		fmt.Println("  --min-score FLOAT  Minimum similarity score (default: 0.6)")
		fmt.Println("  --json             Output as JSON")
		fmt.Println("  --no-summary       Skip meeting summary")
		return
	}

	query := args[0]
	jsonOutput := false
	topN := 10
	minScore := 0.35
	sinceDate := ""
	workspaceID := ""

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--top":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &topN)
				i++
			}
		case "--min-score":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%f", &minScore)
				i++
			}
		case "--since":
			if i+1 < len(args) {
				sinceDate = args[i+1]
				i++
			}
		case "--workspace":
			if i+1 < len(args) {
				workspaceID = args[i+1]
				i++
			}
		}
	}

	// Initialize embeddings
	if err := db.InitEmbeddings(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize embeddings: %v\n", err)
	}

	// Get chunks from database
	chunks, err := db.GetAllChunks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(chunks) == 0 {
		fmt.Println("No embeddings found. Run 'granola embedding backfill' to generate embeddings first.")
		printEmbeddingSetupHint()
		return
	}

	provider := embeddings.NewOllamaProvider()
	engine := embeddings.NewSimilarityEngine(provider)

	// Convert chunks to ChunkData format
	chunkData := make([]embeddings.ChunkData, len(chunks))
	for i, chunk := range chunks {
		chunkData[i] = embeddings.ChunkData{
			ID:           chunk.ID,
			MeetingID:    chunk.MeetingID,
			MeetingTitle: "", // Will be filled from meetings table
			ChunkIndex:   chunk.ChunkIndex,
			ChunkText:    chunk.ChunkText,
			Embedding:    chunk.Embedding,
			Dimensions:   chunk.Dimensions,
			Provider:     chunk.Provider,
			Model:        chunk.Model,
			StartTime:    chunk.StartTime,
			EndTime:      chunk.EndTime,
			Speakers:     chunk.Speakers,
		}
	}

	searchQuery := embeddings.SearchQuery{
		Text:      query,
		MinScore:  minScore,
		Limit:     topN,
		Workspace: workspaceID,
		Since:     sinceDate,
	}

	results, err := engine.Search(context.Background(), searchQuery, chunkData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure Ollama is running with nomic-embed-text:\n")
		fmt.Fprintf(os.Stderr, "  ollama serve\n")
		fmt.Fprintf(os.Stderr, "  ollama pull nomic-embed-text\n")
		os.Exit(1)
	}

	// Build a map of meeting titles and filter by workspace/date
	meetingTitles := make(map[string]string)
	var filteredResults []embeddings.SearchResult

	for _, result := range results {
		// Get meeting title
		title, err := db.GetMeetingTitle(result.MeetingID)
		if err != nil {
			title = "Unknown Meeting"
		}
		meetingTitles[result.MeetingID] = title

		// Filter by workspace if specified
		if workspaceID != "" {
			meeting, err := db.GetMeeting(context.Background(), result.MeetingID)
			if err != nil || meeting.WorkspaceID != workspaceID {
				continue
			}
		}

		// Filter by date if specified
		if sinceDate != "" {
			meeting, err := db.GetMeeting(context.Background(), result.MeetingID)
			if err != nil {
				continue
			}
			// Parse and compare dates properly
			meetingTime, err := time.Parse(time.RFC3339, meeting.CreatedAt)
			if err != nil {
				continue
			}
			parsedDate, err := time.Parse("2006-01-02", sinceDate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid date format for --since, use YYYY-MM-DD\n")
				continue
			}
			if meetingTime.Before(parsedDate) {
				continue
			}
		}

		filteredResults = append(filteredResults, result)
	}

	if len(filteredResults) == 0 {
		fallbackResults := lexicalFallbackResults(query, chunkData, topN)
		for _, result := range fallbackResults {
			title, err := db.GetMeetingTitle(result.MeetingID)
			if err != nil {
				title = "Unknown Meeting"
			}
			meetingTitles[result.MeetingID] = title
			filteredResults = append(filteredResults, result)
		}
	}

	if len(filteredResults) == 0 {
		fmt.Println("No matching results found.")
		fmt.Println("If results are missing, check `granola embedding status` and run `granola embedding backfill`.")
		fmt.Println("See README: Embeddings & Semantic Search for Ollama/model setup details.")
		return
	}

	if jsonOutput {
		output.JSON(filteredResults)
	} else {
		grouped := groupSearchResults(filteredResults, meetingTitles)
		fmt.Printf("Found %d results across %d meetings for \"%s\"\n\n", len(filteredResults), len(grouped), query)
		for i, group := range grouped {
			title := group.MeetingTitle
			if title == "" {
				title = "Unknown Meeting"
			}
			fmt.Printf("%d. %s\n", i+1, title)
			fmt.Printf("   Best Relevance: %.1f%%\n", group.BestScore*100)
			fmt.Printf("   Match Count: %d\n", len(group.Matches))
			for j, result := range group.Matches {
				if j >= 2 {
					break
				}
				fmt.Printf("   Match %d: %s\n", j+1, truncateString(result.ChunkText, 180))
			}
			fmt.Println()
		}
	}
}

func printInferenceSetupHint(err error, cfg *config.Config) {
	if err == nil || cfg == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") || strings.Contains(msg, "timeout") || strings.Contains(msg, "context deadline exceeded") {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Inference backend hint:")
		fmt.Fprintf(os.Stderr, "- Current base_url: %s\n", cfg.BaseURL)
		fmt.Fprintf(os.Stderr, "- Current model: %s\n", cfg.Model)
		fmt.Fprintln(os.Stderr, "- Make sure your local model backend is running (for example: `ollama serve`).")
		fmt.Fprintln(os.Stderr, "- Make sure the requested model is installed and available.")
		fmt.Fprintln(os.Stderr, "- See README: Embeddings & Semantic Search and Configuration.")
	}
}

func printEmbeddingSetupHint() {
	fmt.Println("Embedding setup checklist:")
	fmt.Println("- Granola meeting data available locally (`granola auth login`, `granola meeting list`) ")
	fmt.Println("- Ollama running (`ollama serve`)")
	fmt.Println("- Embedding model installed (`ollama pull nomic-embed-text`)")
	fmt.Println("- Then rerun `granola embedding backfill`")
	fmt.Println("See README: Embeddings & Semantic Search.")
}

func lexicalFallbackResults(query string, chunks []embeddings.ChunkData, limit int) []embeddings.SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	terms := strings.Fields(query)
	type candidate struct {
		result embeddings.SearchResult
		score  float64
	}
	results := make([]candidate, 0)
	for _, chunk := range chunks {
		text := strings.ToLower(chunk.ChunkText)
		title := strings.ToLower(chunk.MeetingTitle)
		matches := 0
		for _, term := range terms {
			if strings.Contains(text, term) || strings.Contains(title, term) {
				matches++
			}
		}
		if matches == 0 {
			continue
		}
		score := float64(matches) / float64(len(terms))
		results = append(results, candidate{result: embeddings.SearchResult{
			MeetingID:    chunk.MeetingID,
			MeetingTitle: chunk.MeetingTitle,
			Score:        score,
			ChunkIndex:   chunk.ChunkIndex,
			ChunkText:    chunk.ChunkText,
			StartTime:    chunk.StartTime,
			EndTime:      chunk.EndTime,
			Speakers:     chunk.Speakers,
			Provider:     chunk.Provider,
			Model:        chunk.Model,
		}, score: score})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	if len(results) > limit {
		results = results[:limit]
	}
	out := make([]embeddings.SearchResult, len(results))
	for i, item := range results {
		out[i] = item.result
	}
	return out
}

func runEmbedding(args []string) {
	if len(args) < 1 {
		fmt.Println("Embedding commands: status, backfill, reset")
		return
	}

	switch args[0] {
	case "status":
		runEmbeddingStatus(args[1:])
	case "backfill":
		runEmbeddingBackfill(args[1:])
	case "reset":
		runEmbeddingReset(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown embedding command: %s\n", args[0])
	}
}

func runEmbeddingStatus(args []string) {
	jsonOutput := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		}
	}

	stats := db.GetEmbeddingStats()

	if jsonOutput {
		output.JSON(stats)
	} else {
		fmt.Println("Embedding Status")
		fmt.Println("================")
		fmt.Printf("Total meetings:     %d\n", stats.Total)
		fmt.Printf("Embedded:           %d\n", stats.Embedded)
		fmt.Printf("Pending:            %d\n", stats.Pending)
		fmt.Printf("Total chunks:       %d\n", stats.TotalChunks)
		fmt.Printf("Coverage:           %d%%\n", stats.Coverage)
	}
}

func runEmbeddingBackfill(args []string) {
	limit := defaultLimit
	batchSize := defaultBatchSize
	dryRun := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &limit)
				i++
			}
		case "--batch":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &batchSize)
				if batchSize <= 0 {
					fmt.Fprintf(os.Stderr, "Warning: Invalid batch size, using default\n")
					batchSize = 10
				}
				i++
			}
		case "--dry-run":
			dryRun = true
		}
	}

	// Initialize embeddings
	if err := db.InitEmbeddings(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize embeddings: %v\n", err)
	}

	meetings, err := db.GetMeetingsNeedingEmbeddings(limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(meetings) == 0 {
		fmt.Println("All meetings already have embeddings.")
		return
	}

	if dryRun {
		fmt.Printf("Dry run: Would process %d meetings\n", len(meetings))
		for _, m := range meetings {
			fmt.Printf("  - %s (%s)\n", m.Title, m.ID)
		}
		return
	}

	provider := embeddings.NewOllamaProvider()
	chunker := embeddings.NewChunker(embeddings.DefaultChunkerConfig())

	processed := 0
	errored := 0

	for i := 0; i < len(meetings); i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > len(meetings) {
			batchEnd = len(meetings)
		}

		batch := meetings[i:batchEnd]
		fmt.Fprintf(os.Stderr, "Processing batch %d/%d...\n", i/batchSize+1, (len(meetings)+batchSize-1)/batchSize)

		for _, meeting := range batch {
			meetingErrors := 0
			// Fetch transcript
			transcript, err := granolaClient.GetTranscript(meeting.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching transcript for %s: %v\n", meeting.ID, err)
				errored++
				continue
			}

			// Format transcript with meeting metadata so semantic search can match
			// titles, attendees, and other high-signal meeting descriptors.
			var transcriptText strings.Builder
			transcriptText.WriteString(fmt.Sprintf("Meeting Title: %s\n", meeting.Title))
			if meeting.Attendees.Valid && strings.TrimSpace(meeting.Attendees.String) != "" {
				transcriptText.WriteString(fmt.Sprintf("Attendees: %s\n", meeting.Attendees.String))
			}
			transcriptText.WriteString("Transcript:\n")
			for _, u := range transcript {
				transcriptText.WriteString(fmt.Sprintf("[%s] %s: %s\n", u.StartTimestamp, u.Source, u.Text))
			}

			// Chunk the transcript
			chunkList := chunker.Chunk(transcriptText.String())

			chunkCount := 0

			// Generate embeddings for each chunk
			for _, chunk := range chunkList {
				saved, errs := embedAndStoreChunk(provider, meeting.ID, chunk, db)
				chunkCount += saved
				meetingErrors += errs
				errored += errs
			}

			// Update meeting status
			status := "complete"
			if chunkCount == 0 || meetingErrors > 0 {
				status = "partial"
			}
			db.UpdateMeetingEmbeddingStatus(meeting.ID, status, provider.Name(), provider.Model(), chunkCount)
			processed++
		}
	}

	fmt.Printf("\nProcessed %d meetings (%d errors)\n", processed, errored)
}

func embedAndStoreChunk(provider embeddings.Provider, meetingID string, chunk embeddings.Chunk, db *storage.DB) (int, int) {
	parts := splitForEmbedding(chunk.Text, provider.MaxChars())
	if len(parts) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: Chunk %d too large (%d chars), splitting into %d parts...\n", chunk.Index, len(chunk.Text), len(parts))
	}
	saved := 0
	errs := 0
	for splitIndex, part := range parts {
		embedding, err := provider.GenerateEmbedding(context.Background(), part)
		if err != nil {
			if isEmbeddingTooLong(err) && len(part) > 250 {
				moreSaved, moreErrs := embedAndStoreChunk(provider, meetingID, embeddings.Chunk{
					Index:     chunk.Index*1000 + splitIndex,
					Text:      part,
					StartTime: chunk.StartTime,
					EndTime:   chunk.EndTime,
					Speakers:  chunk.Speakers,
				}, db)
				saved += moreSaved
				errs += moreErrs
				continue
			}
			fmt.Fprintf(os.Stderr, "Error generating embedding: %v\n", err)
			errs++
			continue
		}

		embeddingChunk := &storage.EmbeddingChunk{
			ID:         fmt.Sprintf("%s:chunk:%d:%d", meetingID, chunk.Index, splitIndex),
			MeetingID:  meetingID,
			ChunkIndex: chunk.Index*1000 + splitIndex,
			ChunkText:  part,
			Embedding:  embeddings.EmbedToBlob(embedding),
			Dimensions: provider.Dimensions(),
			Provider:   provider.Name(),
			Model:      provider.Model(),
			StartTime:  chunk.StartTime,
			EndTime:    chunk.EndTime,
			Speakers:   chunk.Speakers,
			CreatedAt:  time.Now().Unix(),
		}
		if err := db.SaveEmbeddingChunk(embeddingChunk); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving chunk: %v\n", err)
			errs++
			continue
		}
		saved++
	}
	return saved, errs
}

func splitForEmbedding(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxChars <= 0 {
		maxChars = 2000
	}
	if len(text) <= maxChars {
		return []string{text}
	}
	parts := make([]string, 0)
	remaining := text
	for len(remaining) > 0 {
		if len(remaining) <= maxChars {
			parts = append(parts, strings.TrimSpace(remaining))
			break
		}
		splitAt := bestSplitPoint(remaining, maxChars)
		part := strings.TrimSpace(remaining[:splitAt])
		if part != "" {
			parts = append(parts, part)
		}
		remaining = strings.TrimSpace(remaining[splitAt:])
	}
	return parts
}

func bestSplitPoint(text string, maxChars int) int {
	if len(text) <= maxChars {
		return len(text)
	}
	window := text[:maxChars]
	if idx := strings.LastIndex(window, "\n"); idx > maxChars/2 {
		return idx
	}
	for _, sep := range []string{". ", "! ", "? ", "; ", ", ", " "} {
		if idx := strings.LastIndex(window, sep); idx > maxChars/2 {
			return idx + len(sep)
		}
	}
	return maxChars
}

func isEmbeddingTooLong(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context length") || strings.Contains(msg, "input length exceeds")
}

func runEmbeddingReset(args []string) {
	force := false
	for _, arg := range args {
		if arg == "--force" {
			force = true
		}
	}

	if !force {
		fmt.Print("This will delete all embeddings. Continue? (y/N) ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("Cancelled.")
			return
		}
	}

	if err := db.ResetAllEmbeddings(); err != nil {
		fmt.Fprintf(os.Stderr, "Error resetting embeddings: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Embeddings reset successfully.")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// validateAndWriteFile securely writes file content with path validation
func validateAndWriteFile(path string, data []byte, perm os.FileMode) error {
	// Get absolute path and clean it (resolves symlinks, removes ..)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)

	// Get allowed directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/"
	}
	tempDir := os.TempDir()

	// Validate path is within allowed directories
	allowedDirs := []string{homeDir, tempDir}
	valid := false
	for _, allowed := range allowedDirs {
		if cleanPath == allowed || strings.HasPrefix(cleanPath, allowed+string(os.PathSeparator)) {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid path: must be in home directory or temp directory")
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(cleanPath, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
