package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// CodexExecArgs represents the arguments for executing a codex command.
type CodexExecArgs struct {
	Input string

	BaseUrl               string
	ApiKey                string
	ThreadId              *string
	Images                []string
	Model                 string
	SandboxMode           string
	WorkingDirectory      string
	AdditionalDirectories []string
	SkipGitRepoCheck      bool
	DisableSkills         bool
	OutputSchemaFile      string
	ModelReasoningEffort  string
	Context               context.Context
	NetworkAccessEnabled  bool
	WebSearchMode         string
	WebSearchEnabled      *bool
	ApprovalPolicy        string
}

// ExecResult represents a result from executing codex.
type ExecResult struct {
	Line  string
	Error error
}

// CodexExec handles execution of the codex binary.
type CodexExec struct {
	executablePath string
	envOverride    map[string]string
	verbose        bool
	verboseWriter  io.Writer
}

// NewCodexExec creates a new CodexExec instance.
func NewCodexExec(executablePath string, envOverride map[string]string) *CodexExec {
	if executablePath == "" {
		executablePath = findCodexPath()
	}
	return &CodexExec{
		executablePath: executablePath,
		envOverride:    envOverride,
	}
}

// EnableVerbose enables debug logging for Codex exec.
func (c *CodexExec) EnableVerbose(writer io.Writer) {
	c.verbose = true
	if writer != nil {
		c.verboseWriter = writer
	} else {
		c.verboseWriter = os.Stderr
	}
}

func (c *CodexExec) logf(format string, args ...interface{}) {
	if !c.verbose {
		return
	}
	if c.verboseWriter == nil {
		c.verboseWriter = os.Stderr
	}
	fmt.Fprintf(c.verboseWriter, format+"\n", args...)
}

func summarizeEventLine(line string) string {
	var meta struct {
		Type string `json:"type"`
		Item *struct {
			Type string `json:"type"`
			// Common item fields used for debugging output.
			Status   string `json:"status"`
			Command  string `json:"command"`
			ExitCode *int   `json:"exit_code"`
			Server   string `json:"server"`
			Tool     string `json:"tool"`
		} `json:"item"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return fmt.Sprintf("stdout line (%d bytes)", len(line))
	}
	if meta.Type == "" {
		return fmt.Sprintf("stdout line (%d bytes)", len(line))
	}

	summary := fmt.Sprintf("event %s", meta.Type)
	if meta.Item != nil && meta.Item.Type != "" {
		summary = fmt.Sprintf("%s item=%s", summary, meta.Item.Type)
		switch meta.Item.Type {
		case "command_execution":
			if meta.Item.Status != "" {
				summary = fmt.Sprintf("%s status=%s", summary, meta.Item.Status)
			}
			if meta.Item.ExitCode != nil {
				summary = fmt.Sprintf("%s exit=%d", summary, *meta.Item.ExitCode)
			}
			if meta.Item.Command != "" {
				summary = fmt.Sprintf("%s cmd=%q", summary, meta.Item.Command)
			}
		case "mcp_tool_call":
			if meta.Item.Server != "" || meta.Item.Tool != "" {
				summary = fmt.Sprintf("%s mcp=%s/%s", summary, meta.Item.Server, meta.Item.Tool)
			}
		}
	}
	if meta.Error != nil && meta.Error.Message != "" {
		summary = fmt.Sprintf("%s error=%q", summary, meta.Error.Message)
	}
	return summary
}

// Run executes codex with the given arguments and returns a channel of results.
func (c *CodexExec) Run(args CodexExecArgs) <-chan ExecResult {
	output := make(chan ExecResult)

	go func() {
		defer close(output)

		ctx := args.Context
		if ctx == nil {
			ctx = context.Background()
		}

		commandArgs := []string{"exec", "--experimental-json"}

		if args.Model != "" {
			commandArgs = append(commandArgs, "--model", args.Model)
		}

		if args.SandboxMode != "" {
			commandArgs = append(commandArgs, "--sandbox", args.SandboxMode)
		}

		if args.WorkingDirectory != "" {
			commandArgs = append(commandArgs, "--cd", args.WorkingDirectory)
		}

		for _, dir := range args.AdditionalDirectories {
			commandArgs = append(commandArgs, "--add-dir", dir)
		}

		if args.SkipGitRepoCheck {
			commandArgs = append(commandArgs, "--skip-git-repo-check")
		}

		if args.DisableSkills {
			commandArgs = append(commandArgs, "--config", "features.skills=false")
		}

		if args.OutputSchemaFile != "" {
			commandArgs = append(commandArgs, "--output-schema", args.OutputSchemaFile)
		}

		if args.ModelReasoningEffort != "" {
			commandArgs = append(commandArgs, "--config", fmt.Sprintf(`model_reasoning_effort="%s"`, args.ModelReasoningEffort))
		}

		if args.NetworkAccessEnabled {
			commandArgs = append(commandArgs, "--config", fmt.Sprintf(`sandbox_workspace_write.network_access=%t`, args.NetworkAccessEnabled))
		}

		if args.WebSearchMode != "" {
			commandArgs = append(commandArgs, "--config", fmt.Sprintf(`web_search="%s"`, args.WebSearchMode))
		} else if args.WebSearchEnabled != nil {
			if *args.WebSearchEnabled {
				commandArgs = append(commandArgs, "--config", `web_search="live"`)
			} else {
				commandArgs = append(commandArgs, "--config", `web_search="disabled"`)
			}
		}

		if args.ApprovalPolicy != "" {
			commandArgs = append(commandArgs, "--config", fmt.Sprintf(`approval_policy="%s"`, args.ApprovalPolicy))
		}

		for _, image := range args.Images {
			commandArgs = append(commandArgs, "--image", image)
		}

		if args.ThreadId != nil {
			commandArgs = append(commandArgs, "resume", *args.ThreadId)
		}

		c.logf("codex exec: %s %s", c.executablePath, strings.Join(commandArgs, " "))
		if args.ThreadId != nil {
			c.logf("codex exec thread id: %s", *args.ThreadId)
		}
		c.logf("codex exec input bytes: %d", len(args.Input))

		// Set up environment
		env := os.Environ()
		if c.envOverride != nil {
			env = []string{}
			for k, v := range c.envOverride {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		// Check for internal originator override
		foundOriginator := false
		for _, e := range env {
			if strings.HasPrefix(e, "CODEX_INTERNAL_ORIGINATOR_OVERRIDE=") {
				foundOriginator = true
				break
			}
		}
		if !foundOriginator {
			env = append(env, "CODEX_INTERNAL_ORIGINATOR_OVERRIDE=codex_sdk_go")
		}

		// Set API key and base URL
		if args.BaseUrl != "" {
			env = append(env, fmt.Sprintf("OPENAI_BASE_URL=%s", args.BaseUrl))
		}
		if args.ApiKey != "" {
			env = append(env, fmt.Sprintf("CODEX_API_KEY=%s", args.ApiKey))
		}

		// Create command
		cmd := exec.Command(c.executablePath, commandArgs...)
		cmd.Env = env

		// Set up stdin
		stdin, err := cmd.StdinPipe()
		if err != nil {
			output <- ExecResult{Error: fmt.Errorf("failed to create stdin pipe: %w", err)}
			return
		}

		// Set up stdout
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			output <- ExecResult{Error: fmt.Errorf("failed to create stdout pipe: %w", err)}
			return
		}

		// Set up stderr capture
		stderr, err := cmd.StderrPipe()
		if err != nil {
			output <- ExecResult{Error: fmt.Errorf("failed to create stderr pipe: %w", err)}
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			output <- ExecResult{Error: fmt.Errorf("failed to start codex: %w", err)}
			return
		}
		if cmd.Process != nil {
			c.logf("codex exec started (pid %d)", cmd.Process.Pid)
		}

		// Write input to stdin
		go func() {
			defer stdin.Close()
			written, err := stdin.Write([]byte(args.Input))
			if err != nil {
				// Error will be picked up from stderr or exit code
				return
			}
			c.logf("codex exec stdin wrote %d bytes", written)
		}()

		// Capture stderr
		var stderrBuilder strings.Builder
		var stderrWg sync.WaitGroup
		stderrWg.Add(1)
		go func() {
			defer stderrWg.Done()
			reader := bufio.NewReader(stderr)
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					stderrBuilder.WriteString(line)
					c.logf("codex exec stderr: %s", strings.TrimRight(line, "\r\n"))
				}
				if err != nil {
					if err != io.EOF {
						c.logf("codex exec stderr read error: %v", err)
					}
					return
				}
			}
		}()

		// Create a context-aware reader
		done := make(chan struct{})

		go func() {
			defer close(done)
			reader := bufio.NewReader(stdout)
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					line = strings.TrimRight(line, "\r\n")
					if line != "" {
						c.logf("codex exec stdout: %s", summarizeEventLine(line))
					}
					select {
					case output <- ExecResult{Line: line}:
					case <-ctx.Done():
						if cmd.Process != nil {
							_ = cmd.Process.Kill()
						}
						return
					}
				}
				if err != nil {
					if err == io.EOF {
						return
					}
					output <- ExecResult{Error: fmt.Errorf("failed to read codex stdout: %w", err)}
					return
				}
			}
		}()

		// Wait for completion or context cancellation
		select {
		case <-done:
			// Scanner finished
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			output <- ExecResult{Error: ctx.Err()}
			return
		}

		// Wait for stderr
		stderrWg.Wait()

		// Wait for command to finish
		if err := cmd.Wait(); err != nil {
			c.logf("codex exec exited with error: %v", err)
			if exitErr, ok := err.(*exec.ExitError); ok {
				output <- ExecResult{
					Error: fmt.Errorf("codex exited with code %d: %s", exitErr.ExitCode(), stderrBuilder.String()),
				}
			} else {
				output <- ExecResult{Error: fmt.Errorf("codex execution failed: %w", err)}
			}
			return
		}
		c.logf("codex exec completed")
	}()

	return output
}

// findCodexPath finds the path to the codex binary based on platform and architecture.
func findCodexPath() string {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	var targetTriple string

	switch platform {
	case "linux":
		switch arch {
		case "amd64":
			targetTriple = "x86_64-unknown-linux-musl"
		case "arm64":
			targetTriple = "aarch64-unknown-linux-musl"
		default:
			panic(fmt.Sprintf("unsupported architecture on linux: %s", arch))
		}
	case "darwin":
		switch arch {
		case "amd64":
			targetTriple = "x86_64-apple-darwin"
		case "arm64":
			targetTriple = "aarch64-apple-darwin"
		default:
			panic(fmt.Sprintf("unsupported architecture on darwin: %s", arch))
		}
	case "windows":
		switch arch {
		case "amd64":
			targetTriple = "x86_64-pc-windows-msvc"
		case "arm64":
			targetTriple = "aarch64-pc-windows-msvc"
		default:
			panic(fmt.Sprintf("unsupported architecture on windows: %s", arch))
		}
	default:
		panic(fmt.Sprintf("unsupported platform: %s (%s)", platform, arch))
	}

	// First, try to find the codex binary relative to the currently running binary
	// This handles the case where codex-orchestrator is installed in $GOPATH/bin
	if execPath, err := os.Executable(); err == nil {
		// Get the directory where the running binary is located
		binDir := filepath.Dir(execPath)

		// Try to find vendor directory in parent directories
		// Look for patterns like:
		// - $GOPATH/bin/codex-orchestrator -> $GOPATH/src/github.com/.../codex-go-sdk/vendor
		// - ./bin/codex-orchestrator -> ./vendor
		// - ./codex-orchestrator -> ./vendor

		// Search parent directories for vendor folder
		currentDir := binDir
		for i := 0; i < 5; i++ { // Search up to 5 levels
			// Try vendor at this level
			vendorRoot := filepath.Join(currentDir, "vendor")
			archRoot := filepath.Join(vendorRoot, targetTriple)
			binaryName := "codex"
			if platform == "windows" {
				binaryName = "codex.exe"
			}
			binaryPath := filepath.Join(archRoot, "codex", binaryName)

			if _, err := os.Stat(binaryPath); err == nil {
				return binaryPath
			}

			// Move up one directory
			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir {
				break // Reached filesystem root
			}
			currentDir = parentDir
		}

		// Also try relative to bin directory for local development
		// ./bin/codex-orchestrator -> ../codex-go-sdk/vendor
		vendorRoot := filepath.Join(binDir, "..", "codex-go-sdk", "vendor")
		archRoot := filepath.Join(vendorRoot, targetTriple)
		binaryName := "codex"
		if platform == "windows" {
			binaryName = "codex.exe"
		}
		binaryPath := filepath.Join(archRoot, "codex", binaryName)

		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath
		}
	}

	// Fallback 1: Try relative to current working directory (original behavior)
	vendorRoot := filepath.Join("..", "codex-go-sdk", "vendor")
	archRoot := filepath.Join(vendorRoot, targetTriple)
	binaryName := "codex"
	if platform == "windows" {
		binaryName = "codex.exe"
	}
	binaryPath := filepath.Join(archRoot, "codex", binaryName)

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Fallback 2: Try in ~/.codex-orchestrator/vendor
	homeDir, err := os.UserHomeDir()
	if err == nil {
		vendorRoot := filepath.Join(homeDir, ".codex-orchestrator", "vendor")
		archRoot := filepath.Join(vendorRoot, targetTriple)
		binaryName := "codex"
		if platform == "windows" {
			binaryName = "codex.exe"
		}
		binaryPath := filepath.Join(archRoot, "codex", binaryName)

		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath
		}
	}

	// Fallback 3: Try to find codex in PATH
	lookupName := "codex"
	if platform == "windows" {
		lookupName = "codex.exe"
	}
	if path, err := exec.LookPath(lookupName); err == nil {
		return path
	}

	// Last resort: return the default relative path even if it doesn't exist
	// The error will be handled when trying to execute it
	return binaryPath
}
