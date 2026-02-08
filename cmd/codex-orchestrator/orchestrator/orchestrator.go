package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fanwenlin/codex-go-sdk/codex"
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
}

// OrchestratorResult contains the result of running the orchestrator.
type OrchestratorResult struct {
	FinalResponse string
	Items         []interface{}
}

const (
	DefaultMaxFileBytes  = 256 * 1024
	DefaultMaxTotalBytes = 2 * 1024 * 1024
)

var DefaultIgnoreDirs = []string{
	".git",
	"node_modules",
	"dist",
	"build",
	"coverage",
	".next",
}

var DefaultPreamble = []string{
	"You are a professional coding agent.",
	"Use the provided documents and skills as the source of truth for requirements,",
	"background, and acceptance criteria.",
	"The Skills section contains behavioral instructions you must follow.",
	"Complete the work and respond with your final answer only.",
}

// CollectDocumentBundle collects documents and skills from the specified directories.
func CollectDocumentBundle(options OrchestratorOptions) (*DocumentBundle, error) {
	// Set defaults
	if options.IncludeSkills == false && options.IncludeSkills != true {
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
		if stat, err := os.Stat(resolvedSkillsDir); err == nil && stat.IsDir() {
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
		skillsRel, err := filepath.Rel(resolvedDocDir, resolvedSkillsDir)
		if err == nil {
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

// RunOrchestrator runs the orchestrator with the given options.
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

	turn, err := thread.Run(prompt, codex.TurnOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to run codex: %w", err)
	}

	// Convert ThreadItem slice to interface{} slice
	items := make([]interface{}, len(turn.Items))
	for i, item := range turn.Items {
		items[i] = item
	}

	return &OrchestratorResult{
		FinalResponse: turn.FinalResponse,
		Items:         items,
	}, nil
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

			nested, err := walkDir(rootDir, entryPath, options)
			if err != nil {
				return nil, err
			}
			results = append(results, nested...)
			continue
		}

		// Skip non-regular files
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}

		// Read file
		//nolint:gosec // Reading files from user-provided directory is expected behavior
		content, err := os.ReadFile(entryPath)
		if err != nil {
			continue
		}

		// Check if it's text
		if !isProbablyText(content) {
			continue
		}

		// Apply size limit
		includeBytes := min(len(content), options.maxFileBytes)
		if options.state.totalBytes+includeBytes > options.state.maxTotalBytes {
			options.state.hitLimit = true
			break
		}

		fileContent := string(content[:includeBytes])
		if len(content) > includeBytes {
			fileContent = fmt.Sprintf("%s\n[truncated %d bytes]", fileContent, len(content)-includeBytes)
		}

		relativePath, err := filepath.Rel(rootDir, entryPath)
		if err != nil {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
