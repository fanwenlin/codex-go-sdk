package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fanwenlin/codex-go-sdk/codex"
	"github.com/fanwenlin/codex-go-sdk/types"
)

// DocumentEntry represents a document entry.
type DocumentEntry struct {
	Path    string
	Content string
	Bytes   int
}

// DocumentBundle contains documents and skills.
type DocumentBundle struct {
	Documents  []DocumentEntry
	Skills     []DocumentEntry
	TotalBytes int
}

// OrchestratorOptions contains options for the orchestrator.
//
//nolint:revive // Name stutter is acceptable for exported API
type OrchestratorOptions struct {
	DocDir              string
	IncludeSkills       bool
	DisableGlobalSkills bool
	SkillsDir           string
	MaxFileBytes        int
	MaxTotalBytes       int
	IgnoreDirNames      []string
	PromptPreamble      string
	Verbose             bool
	VerboseWriter       io.Writer
	ProgressWriter      io.Writer // Writer for progress output with timestamps
}

// OrchestratorResult contains the result of running the orchestrator.
//
//nolint:revive // Name stutter is acceptable for exported API
type OrchestratorResult struct {
	FinalResponse string
	Items         []interface{}
}

const (
	// DefaultMaxFileBytes is the default per-file read limit in bytes.
	DefaultMaxFileBytes = 256 * 1024
	// DefaultMaxTotalBytes is the default total read limit in bytes.
	DefaultMaxTotalBytes = 2 * 1024 * 1024

	// Output formatting constants.
	statusOK           = "✓"
	statusError        = "✗"
	statusDeclined     = "⊘"
	maxCommandPreview  = 60
	maxQueryPreview    = 50
	maxResponsePreview = 50
	maxOutputLines     = 10
)

// DefaultIgnoreDirs lists directory names skipped during document traversal.
//
//nolint:gochecknoglobals // Default configuration values
var DefaultIgnoreDirs = []string{
	".git",
	"node_modules",
	"dist",
	"build",
	"coverage",
	".next",
}

// DefaultPreamble contains the default prompt preamble text.
//
//nolint:gochecknoglobals // Default configuration values
var DefaultPreamble = []string{
	"You are a professional coding agent.",
	"Use the provided documents and skills as the source of truth for requirements,",
	"background, and acceptance criteria.",
	"The Skills section contains behavioral instructions you must follow.",
	"Complete the work and respond with your final answer only.",
	"Don't forget to run lint and unit tests locally after coding to verify changes",
}

// CollectDocumentBundle collects documents and skills from the specified directories.
//
//nolint:gocognit // Document collection logic is inherently sequential
func CollectDocumentBundle(options OrchestratorOptions) (*DocumentBundle, error) {
	// Set defaults
	if !options.IncludeSkills {
		options.IncludeSkills = true
	}
	if options.MaxFileBytes == 0 {
		options.MaxFileBytes = DefaultMaxFileBytes
	}
	if options.MaxTotalBytes == 0 {
		options.MaxTotalBytes = DefaultMaxTotalBytes
	}
	if len(options.IgnoreDirNames) == 0 {
		options.IgnoreDirNames = DefaultIgnoreDirs
	}

	// Resolve document directory
	resolvedDocDir, err := filepath.Abs(options.DocDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve document directory: %w", err)
	}

	// Check if document directory exists and is a directory
	stat, err := os.Stat(resolvedDocDir)
	if err != nil {
		return nil, fmt.Errorf("document directory not found: %s", options.DocDir)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("document directory is not a directory: %s", options.DocDir)
	}

	// Resolve skills directory
	var resolvedSkillsDir string
	if options.SkillsDir != "" {
		resolvedSkillsDir, err = filepath.Abs(options.SkillsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve skills directory: %w", err)
		}
	} else if options.IncludeSkills {
		resolvedSkillsDir = filepath.Join(resolvedDocDir, "skills")
	}

	state := &walkState{
		totalBytes:    0,
		maxTotalBytes: options.MaxTotalBytes,
		hitLimit:      false,
	}

	// Collect skills first
	var skills []DocumentEntry
	if resolvedSkillsDir != "" {
		if skillsStat, statErr := os.Stat(resolvedSkillsDir); statErr == nil && skillsStat.IsDir() {
			skills, err = walkDir(resolvedSkillsDir, resolvedSkillsDir, walkOptions{
				ignoreDirNames: options.IgnoreDirNames,
				maxFileBytes:   options.MaxFileBytes,
				state:          state,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to collect skills: %w", err)
			}
		}
	}

	// Collect documents
	documents, err := walkDir(resolvedDocDir, resolvedDocDir, walkOptions{
		ignoreDirNames: options.IgnoreDirNames,
		maxFileBytes:   options.MaxFileBytes,
		state:          state,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect documents: %w", err)
	}

	// If skills directory is a subdirectory of document directory, filter out skills from documents
	if resolvedSkillsDir != "" && isSubpath(resolvedSkillsDir, resolvedDocDir) {
		skillsRel, relErr := filepath.Rel(resolvedDocDir, resolvedSkillsDir)
		if relErr == nil {
			skillsRel = normalizeRelPath(skillsRel)
			documents = filterDocuments(documents, skillsRel)
		}
	}

	// Sort by path for consistency
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].Path < documents[j].Path
	})
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Path < skills[j].Path
	})

	return &DocumentBundle{
		Documents:  documents,
		Skills:     skills,
		TotalBytes: state.totalBytes,
	}, nil
}

// BuildPrompt builds the prompt from a document bundle.
func BuildPrompt(bundle *DocumentBundle, preamble string) string {
	if preamble == "" {
		preamble = strings.Join(DefaultPreamble, " ")
	}

	var lines []string
	lines = append(lines, strings.TrimSpace(preamble))

	if len(bundle.Skills) > 0 {
		lines = append(lines, "", "Skills:")
		lines = append(lines, formatEntries(bundle.Skills)...)
	}

	lines = append(lines, "", "Documents:")
	if len(bundle.Documents) > 0 {
		lines = append(lines, formatEntries(bundle.Documents)...)
	} else {
		lines = append(lines, "[No documents found]")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// RunOrchestrator runs the orchestrator with the given options using streaming mode.
func RunOrchestrator(options OrchestratorOptions) (*OrchestratorResult, error) {
	// Collect document bundle
	bundle, err := CollectDocumentBundle(options)
	if err != nil {
		return nil, fmt.Errorf("failed to collect documents: %w", err)
	}

	// Build prompt
	prompt := BuildPrompt(bundle, options.PromptPreamble)

	// Create codex client and run
	codexClient := codex.NewCodex(codex.CodexOptions{
		Verbose:       options.Verbose,
		VerboseWriter: options.VerboseWriter,
	})

	thread := codexClient.StartThread(codex.ThreadOptions{
		SandboxMode:   codex.SandboxModeFullAccess,
		DisableSkills: options.DisableGlobalSkills,
	})

	// Use streaming mode for progress tracking
	stream, err := thread.RunStreamed(prompt, codex.TurnOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start codex stream: %w", err)
	}

	// Process events and track progress
	progressWriter := options.ProgressWriter
	if progressWriter == nil {
		progressWriter = io.Discard
	}

	return processStream(stream.Events, progressWriter)
}

// processStream processes the event stream and returns the final result.
func processStream(events <-chan types.ThreadEvent, writer io.Writer) (*OrchestratorResult, error) {
	var items []interface{}
	var finalResponse string
	var usage *types.Usage
	var turnFailure error

	for event := range events {
		printEventSummary(event, writer)

		switch e := event.(type) {
		case *types.ItemCompletedEvent:
			if agentMsg, ok := e.Item.(*types.AgentMessageItem); ok {
				finalResponse = agentMsg.Text
			}
			items = append(items, e.Item)
		case *types.TurnCompletedEvent:
			usage = &e.Usage
		case *types.TurnFailedEvent:
			turnFailure = fmt.Errorf("turn failed: %s", e.Error.Message)
		case *types.ThreadErrorEvent:
			turnFailure = fmt.Errorf("thread error: %s", e.Message)
		}

		if turnFailure != nil {
			break
		}
	}

	if turnFailure != nil {
		return nil, turnFailure
	}

	// Print summary footer
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(writer, "\n[%s] ✓ Completed", timestamp)
	if usage != nil {
		fmt.Fprintf(writer, " | Tokens: %d in / %d out", usage.InputTokens, usage.OutputTokens)
	}
	fmt.Fprintln(writer)

	return &OrchestratorResult{
		FinalResponse: finalResponse,
		Items:         items,
	}, nil
}

// printEventSummary prints a human-readable summary of an event with timestamp.
func printEventSummary(event types.ThreadEvent, writer io.Writer) {
	timestamp := time.Now().Format("15:04:05")

	switch e := event.(type) {
	case *types.ThreadStartedEvent:
		fmt.Fprintf(writer, "[%s] ▶ Thread started: %s\n", timestamp, e.ThreadId)

	case *types.TurnStartedEvent:
		fmt.Fprintf(writer, "[%s] ▶ Turn started\n", timestamp)

	case *types.ItemStartedEvent:
		printItemStarted(e.Item, timestamp, writer)

	case *types.ItemUpdatedEvent:
		printItemUpdated(e.Item, timestamp, writer)

	case *types.ItemCompletedEvent:
		printItemCompleted(e.Item, timestamp, writer)

	case *types.TurnCompletedEvent:
		// Summary printed at the end

	case *types.TurnFailedEvent:
		fmt.Fprintf(writer, "[%s] ✗ Turn failed: %s\n", timestamp, e.Error.Message)

	case *types.ThreadErrorEvent:
		fmt.Fprintf(writer, "[%s] ✗ Error: %s\n", timestamp, e.Message)
	}
}

// printItemStarted prints a summary for item started event.
func printItemStarted(item types.ThreadItem, timestamp string, writer io.Writer) {
	switch i := item.(type) {
	case *types.CommandExecutionItem:
		fmt.Fprintf(writer, "[%s] $ Executing: %s\n", timestamp, truncate(i.Command, maxCommandPreview))
	case *types.FileChangeItem:
		files := make([]string, 0, len(i.Changes))
		for _, change := range i.Changes {
			files = append(files, change.Path)
		}
		fmt.Fprintf(writer, "[%s] ✎ Modifying %d file(s): %s\n", timestamp, len(i.Changes), strings.Join(files, ", "))
	case *types.McpToolCallItem:
		fmt.Fprintf(writer, "[%s] 🔧 Tool call: %s.%s\n", timestamp, i.Server, i.Tool)
	case *types.AgentMessageItem:
		// Skip, will show on completion
	case *types.ReasoningItem:
		fmt.Fprintf(writer, "[%s] 💭 Reasoning...\n", timestamp)
	case *types.WebSearchItem:
		fmt.Fprintf(writer, "[%s] 🔍 Searching: %s\n", timestamp, truncate(i.Query, maxQueryPreview))
	case *types.TodoListItem:
		active := 0
		for _, todo := range i.Items {
			if !todo.Completed {
				active++
			}
		}
		fmt.Fprintf(writer, "[%s] ☑ Todo list: %d active, %d completed\n", timestamp, active, len(i.Items)-active)
	}
}

// printItemUpdated prints a summary for item updated event.
func printItemUpdated(item types.ThreadItem, timestamp string, writer io.Writer) {
	if i, ok := item.(*types.CommandExecutionItem); ok {
		if i.Status == types.CommandExecutionStatusInProgress && i.AggregatedOutput != nil {
			// Only show output summary on significant updates
			output := *i.AggregatedOutput
			lines := strings.Split(output, "\n")
			if len(lines) > maxOutputLines {
				fmt.Fprintf(writer, "[%s]   ... %d lines of output ...\n", timestamp, len(lines))
			}
		}
	}
}

// printItemCompleted prints a summary for item completed event.
func printItemCompleted(item types.ThreadItem, timestamp string, writer io.Writer) {
	switch i := item.(type) {
	case *types.CommandExecutionItem:
		status := statusOK
		if i.ExitCode != nil && *i.ExitCode != 0 {
			status = fmt.Sprintf("%s (exit %d)", statusError, *i.ExitCode)
		} else if i.Status == types.CommandExecutionStatusFailed {
			status = statusError
		}
		duration := ""
		if i.AggregatedOutput != nil {
			lines := strings.Count(*i.AggregatedOutput, "\n")
			duration = fmt.Sprintf(" | %d lines output", lines+1)
		}
		fmt.Fprintf(writer, "[%s]   %s Command completed%s\n", timestamp, status, duration)

	case *types.FileChangeItem:
		status := statusOK
		switch i.Status {
		case types.PatchApplyStatusFailed:
			status = statusError
		case types.PatchApplyStatusDeclined:
			status = statusDeclined
		case types.PatchApplyStatusInProgress, types.PatchApplyStatusCompleted:
		}
		fmt.Fprintf(writer, "[%s]   %s Files modified (%d changes)\n", timestamp, status, len(i.Changes))

	case *types.McpToolCallItem:
		status := statusOK
		if i.Status == types.McpToolCallStatusFailed {
			status = statusError
		}
		fmt.Fprintf(writer, "[%s]   %s Tool completed: %s.%s\n", timestamp, status, i.Server, i.Tool)

	case *types.AgentMessageItem:
		lines := strings.Split(i.Text, "\n")
		preview := truncate(strings.TrimSpace(lines[0]), maxResponsePreview)
		fmt.Fprintf(writer, "[%s] ← Response: %s\n", timestamp, preview)

	case *types.ReasoningItem:
		fmt.Fprintf(writer, "[%s]   ✓ Reasoning complete (%d points)\n", timestamp, len(i.Summary))

	case *types.WebSearchItem:
		fmt.Fprintf(writer, "[%s]   ✓ Search completed\n", timestamp)

	case *types.TodoListItem:
		completed := 0
		for _, todo := range i.Items {
			if todo.Completed {
				completed++
			}
		}
		fmt.Fprintf(writer, "[%s]   ✓ Todo list updated (%d/%d completed)\n", timestamp, completed, len(i.Items))
	}
}

// truncate truncates a string to maxLen and adds ellipsis if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

type walkOptions struct {
	ignoreDirNames []string
	maxFileBytes   int
	state          *walkState
}

type walkState struct {
	totalBytes    int
	maxTotalBytes int
	hitLimit      bool
}

//nolint:gocognit // Directory walking requires sequential steps
func walkDir(rootDir string, currentDir string, options walkOptions) ([]DocumentEntry, error) {
	if options.state.hitLimit {
		return nil, nil
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", currentDir, err)
	}

	// Sort entries for consistency
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var results []DocumentEntry

	for _, entry := range entries {
		if options.state.hitLimit {
			break
		}

		entryPath := filepath.Join(currentDir, entry.Name())

		if entry.IsDir() {
			// Check if directory should be ignored
			shouldSkip := false
			for _, ignoreName := range options.ignoreDirNames {
				if entry.Name() == ignoreName {
					shouldSkip = true
					break
				}
			}
			if shouldSkip {
				continue
			}

			nested, walkErr := walkDir(rootDir, entryPath, options)
			if walkErr != nil {
				return nil, walkErr
			}
			results = append(results, nested...)
			continue
		}

		// Skip non-regular files
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}

		// Read file
		content, readErr := os.ReadFile(entryPath)
		if readErr != nil {
			continue
		}

		// Check if it's text
		if !isProbablyText(content) {
			continue
		}

		// Apply size limit
		includeBytes := minInt(len(content), options.maxFileBytes)
		if options.state.totalBytes+includeBytes > options.state.maxTotalBytes {
			options.state.hitLimit = true
			break
		}

		fileContent := string(content[:includeBytes])
		if len(content) > includeBytes {
			fileContent = fmt.Sprintf("%s\n[truncated %d bytes]", fileContent, len(content)-includeBytes)
		}

		relativePath, relErr := filepath.Rel(rootDir, entryPath)
		if relErr != nil {
			continue
		}
		relativePath = normalizeRelPath(relativePath)

		bytes := len(fileContent)
		results = append(results, DocumentEntry{
			Path:    relativePath,
			Content: fileContent,
			Bytes:   bytes,
		})
		options.state.totalBytes += bytes
	}

	return results, nil
}

func formatEntries(entries []DocumentEntry) []string {
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("--- %s ---", entry.Path))
		lines = append(lines, strings.TrimRight(entry.Content, "\n"))
		lines = append(lines, "")
	}
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func normalizeRelPath(path string) string {
	return filepath.ToSlash(path)
}

func isSubpath(candidate string, parent string) bool {
	rel, err := filepath.Rel(parent, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

func filterDocuments(documents []DocumentEntry, skillsRel string) []DocumentEntry {
	var filtered []DocumentEntry
	for _, doc := range documents {
		if doc.Path == skillsRel || strings.HasPrefix(doc.Path, skillsRel+"/") {
			continue
		}
		filtered = append(filtered, doc)
	}
	return filtered
}

func isProbablyText(content []byte) bool {
	return bytes.IndexByte(content, 0) == -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
