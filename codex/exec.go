package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fanwenlin/codex-go-sdk/types"
)

const (
	envInternalOriginatorOverrideKey = "CODEX_INTERNAL_ORIGINATOR_OVERRIDE"
	envBaseURLKey                    = "OPENAI_BASE_URL"
	envCodexAPIEnvVar                = "CODEX_API_KEY"
)

// CodexExecArgs represents the arguments for executing a codex command.
type CodexExecArgs struct {
	Input string
	// InputItems preserves structured input for transports that support it.
	InputItems []types.UserInput

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
	ApprovalHandler       types.ApprovalHandler
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

type eventSummaryMeta struct {
	Type string `json:"type"`
	Item *struct {
		Type string `json:"type"`
		// Common item fields used for debugging output.
		Status   string `json:"status"`
		Command  string `json:"command"`
		ExitCode *int   `json:"exitCode"`
		Server   string `json:"server"`
		Tool     string `json:"tool"`
	} `json:"item"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func summarizeEventLine(line string) string {
	meta, ok := parseEventSummaryMeta(line)
	if !ok || meta.Type == "" {
		return fmt.Sprintf("stdout line (%d bytes)", len(line))
	}

	summary := fmt.Sprintf("event %s", meta.Type)
	summary = appendItemSummary(summary, meta.Item)
	summary = appendErrorSummary(summary, meta.Error)
	return summary
}

func parseEventSummaryMeta(line string) (eventSummaryMeta, bool) {
	var meta eventSummaryMeta
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return meta, false
	}
	return meta, true
}

func appendItemSummary(summary string, item *struct {
	Type string `json:"type"`
	// Common item fields used for debugging output.
	Status   string `json:"status"`
	Command  string `json:"command"`
	ExitCode *int   `json:"exitCode"`
	Server   string `json:"server"`
	Tool     string `json:"tool"`
}) string {
	if item == nil || item.Type == "" {
		return summary
	}
	summary = fmt.Sprintf("%s item=%s", summary, item.Type)
	switch item.Type {
	case "commandExecution":
		if item.Status != "" {
			summary = fmt.Sprintf("%s status=%s", summary, item.Status)
		}
		if item.ExitCode != nil {
			summary = fmt.Sprintf("%s exit=%d", summary, *item.ExitCode)
		}
		if item.Command != "" {
			summary = fmt.Sprintf("%s cmd=%q", summary, item.Command)
		}
	case "mcpToolCall":
		if item.Server != "" || item.Tool != "" {
			summary = fmt.Sprintf("%s mcp=%s/%s", summary, item.Server, item.Tool)
		}
	}
	return summary
}

func appendErrorSummary(summary string, metaError *struct {
	Message string `json:"message"`
}) string {
	if metaError == nil || metaError.Message == "" {
		return summary
	}
	return fmt.Sprintf("%s error=%q", summary, metaError.Message)
}

// Run executes codex with the given arguments and returns a channel of results.
func (c *CodexExec) Run(args CodexExecArgs) <-chan ExecResult {
	output := make(chan ExecResult)

	go func() {
		defer close(output)

		ctx := resolveContext(args.Context)
		commandArgs := buildCommandArgs(args)

		c.logf("codex exec: %s %s", c.executablePath, strings.Join(commandArgs, " "))
		if args.ThreadId != nil {
			c.logf("codex exec thread id: %s", *args.ThreadId)
		}
		c.logf("codex exec input bytes: %d", len(args.Input))

		env := buildExecEnv(c.envOverride, args)

		// Create command.
		// #nosec G204 -- Executable path and args are user-provided by design in SDK integrations.
		cmd := exec.CommandContext(ctx, c.executablePath, commandArgs...)
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
		if startErr := cmd.Start(); startErr != nil {
			output <- ExecResult{Error: fmt.Errorf("failed to start codex: %w", startErr)}
			return
		}
		if cmd.Process != nil {
			c.logf("codex exec started (pid %d)", cmd.Process.Pid)
		}

		c.startInputWriter(stdin, args.Input)
		stderrBuilder, stderrWg := c.startStderrCapture(stderr)
		done := c.startStdoutReader(ctx, stdout, output, cmd)

		if waitErr := waitForCompletion(ctx, done, cmd, output); waitErr != nil {
			return
		}

		stderrWg.Wait()

		if waitErr := c.waitForCommand(cmd, stderrBuilder, output); waitErr != nil {
			return
		}
		c.logf("codex exec completed")
	}()

	return output
}

func resolveContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func buildCommandArgs(args CodexExecArgs) []string {
	commandArgs := []string{"exec", "--experimental-json"}
	commandArgs = appendModelArgs(commandArgs, args)
	commandArgs = appendSandboxArgs(commandArgs, args)
	commandArgs = appendWorkingDirArgs(commandArgs, args)
	commandArgs = appendAdditionalDirArgs(commandArgs, args)
	commandArgs = appendFeatureArgs(commandArgs, args)
	commandArgs = appendConfigArgs(commandArgs, args)
	commandArgs = appendImageArgs(commandArgs, args)
	commandArgs = appendThreadArgs(commandArgs, args)
	return commandArgs
}

func appendModelArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.Model != "" {
		commandArgs = append(commandArgs, "--model", args.Model)
	}
	return commandArgs
}

func appendSandboxArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.SandboxMode != "" {
		commandArgs = append(commandArgs, "--sandbox", args.SandboxMode)
	}
	return commandArgs
}

func appendWorkingDirArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.WorkingDirectory != "" {
		commandArgs = append(commandArgs, "--cd", args.WorkingDirectory)
	}
	return commandArgs
}

func appendAdditionalDirArgs(commandArgs []string, args CodexExecArgs) []string {
	for _, dir := range args.AdditionalDirectories {
		commandArgs = append(commandArgs, "--add-dir", dir)
	}
	return commandArgs
}

func appendFeatureArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.SkipGitRepoCheck {
		commandArgs = append(commandArgs, "--skip-git-repo-check")
	}
	if args.DisableSkills {
		commandArgs = append(commandArgs, "--config", "features.skills=false")
	}
	if args.OutputSchemaFile != "" {
		commandArgs = append(commandArgs, "--output-schema", args.OutputSchemaFile)
	}
	return commandArgs
}

func appendConfigArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.ModelReasoningEffort != "" {
		commandArgs = append(
			commandArgs,
			"--config",
			fmt.Sprintf(`model_reasoning_effort="%s"`, args.ModelReasoningEffort),
		)
	}
	if args.NetworkAccessEnabled {
		commandArgs = append(
			commandArgs,
			"--config",
			fmt.Sprintf(`sandbox_workspace_write.network_access=%t`, args.NetworkAccessEnabled),
		)
	}
	commandArgs = appendWebSearchArgs(commandArgs, args)
	if args.ApprovalPolicy != "" {
		commandArgs = append(commandArgs, "--config", fmt.Sprintf(`approval_policy="%s"`, args.ApprovalPolicy))
	}
	return commandArgs
}

func appendWebSearchArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.WebSearchMode != "" {
		return append(commandArgs, "--config", fmt.Sprintf(`web_search="%s"`, args.WebSearchMode))
	}
	if args.WebSearchEnabled == nil {
		return commandArgs
	}
	if *args.WebSearchEnabled {
		return append(commandArgs, "--config", `web_search="live"`)
	}
	return append(commandArgs, "--config", `web_search="disabled"`)
}

func appendImageArgs(commandArgs []string, args CodexExecArgs) []string {
	for _, image := range args.Images {
		commandArgs = append(commandArgs, "--image", image)
	}
	return commandArgs
}

func appendThreadArgs(commandArgs []string, args CodexExecArgs) []string {
	if args.ThreadId != nil {
		commandArgs = append(commandArgs, "resume", *args.ThreadId)
	}
	return commandArgs
}

func buildExecEnv(envOverride map[string]string, args CodexExecArgs) []string {
	env := os.Environ()
	if envOverride != nil {
		env = []string{}
		for k, v := range envOverride {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if !hasOriginatorOverride(env) {
		env = append(env, envInternalOriginatorOverrideKey+"=codex_sdk_go")
	}

	if args.BaseUrl != "" {
		env = append(env, fmt.Sprintf("%s=%s", envBaseURLKey, args.BaseUrl))
	}
	if args.ApiKey != "" {
		env = append(env, fmt.Sprintf("%s=%s", envCodexAPIEnvVar, args.ApiKey))
	}
	return env
}

func hasOriginatorOverride(env []string) bool {
	for _, entry := range env {
		if strings.HasPrefix(entry, envInternalOriginatorOverrideKey+"=") {
			return true
		}
	}
	return false
}

func (c *CodexExec) startInputWriter(stdin io.WriteCloser, input string) {
	go func() {
		defer stdin.Close()
		written, writeErr := stdin.Write([]byte(input))
		if writeErr != nil {
			// Error will be picked up from stderr or exit code.
			return
		}
		c.logf("codex exec stdin wrote %d bytes", written)
	}()
}

func (c *CodexExec) startStderrCapture(stderr io.Reader) (*strings.Builder, *sync.WaitGroup) {
	var stderrBuilder strings.Builder
	var stderrWg sync.WaitGroup
	stderrWg.Add(1)
	go func() {
		defer stderrWg.Done()
		reader := bufio.NewReader(stderr)
		for {
			line, readErr := reader.ReadString('\n')
			if len(line) > 0 {
				stderrBuilder.WriteString(line)
				c.logf("codex exec stderr: %s", strings.TrimRight(line, "\r\n"))
			}
			if readErr != nil {
				if readErr != io.EOF {
					c.logf("codex exec stderr read error: %v", readErr)
				}
				return
			}
		}
	}()
	return &stderrBuilder, &stderrWg
}

func (c *CodexExec) startStdoutReader(
	ctx context.Context,
	stdout io.Reader,
	output chan ExecResult,
	cmd *exec.Cmd,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		reader := bufio.NewReader(stdout)
		for {
			line, readErr := reader.ReadString('\n')
			if len(line) > 0 {
				line = strings.TrimRight(line, "\r\n")
				if line != "" {
					c.logf("codex exec stdout: %s", summarizeEventLine(line))
				}
				select {
				case output <- ExecResult{Line: line}:
				case <-ctx.Done():
					killProcess(cmd)
					return
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					return
				}
				output <- ExecResult{Error: fmt.Errorf("failed to read codex stdout: %w", readErr)}
				return
			}
		}
	}()
	return done
}

func waitForCompletion(ctx context.Context, done <-chan struct{}, cmd *exec.Cmd, output chan ExecResult) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		killProcess(cmd)
		output <- ExecResult{Error: ctx.Err()}
		return ctx.Err()
	}
}

func (c *CodexExec) waitForCommand(cmd *exec.Cmd, stderrBuilder *strings.Builder, output chan ExecResult) error {
	if err := cmd.Wait(); err != nil {
		c.logf("codex exec exited with error: %v", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			output <- ExecResult{
				Error: fmt.Errorf("codex exited with code %d: %s", exitErr.ExitCode(), stderrBuilder.String()),
			}
		} else {
			output <- ExecResult{Error: fmt.Errorf("codex execution failed: %w", err)}
		}
		return err
	}
	return nil
}

func killProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// findCodexPath finds the path to the codex binary based on platform and architecture.
func findCodexPath() string {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	targetTriple := resolveTargetTriple(platform, arch)
	binaryName := resolveBinaryName(platform)

	if path, ok := findBinaryFromExecutable(targetTriple, binaryName); ok {
		return path
	}
	if path, ok := findBinaryFromCwd(targetTriple, binaryName); ok {
		return path
	}
	if path, ok := findBinaryFromHome(targetTriple, binaryName); ok {
		return path
	}
	if path, ok := findBinaryInPath(binaryName); ok {
		return path
	}

	// Last resort: return the default relative path even if it doesn't exist.
	// The error will be handled when trying to execute it.
	return vendorBinaryPath(filepath.Join("..", "codex-go-sdk", "vendor"), targetTriple, binaryName)
}

func resolveTargetTriple(platform, arch string) string {
	switch platform {
	case "linux":
		switch arch {
		case "amd64":
			return "x86_64-unknown-linux-musl"
		case "arm64":
			return "aarch64-unknown-linux-musl"
		default:
			panic(fmt.Sprintf("unsupported architecture on linux: %s", arch))
		}
	case "darwin":
		switch arch {
		case "amd64":
			return "x86_64-apple-darwin"
		case "arm64":
			return "aarch64-apple-darwin"
		default:
			panic(fmt.Sprintf("unsupported architecture on darwin: %s", arch))
		}
	case "windows":
		switch arch {
		case "amd64":
			return "x86_64-pc-windows-msvc"
		case "arm64":
			return "aarch64-pc-windows-msvc"
		default:
			panic(fmt.Sprintf("unsupported architecture on windows: %s", arch))
		}
	default:
		panic(fmt.Sprintf("unsupported platform: %s (%s)", platform, arch))
	}
}

func resolveBinaryName(platform string) string {
	if platform == "windows" {
		return "codex.exe"
	}
	return "codex"
}

func findBinaryFromExecutable(targetTriple, binaryName string) (string, bool) {
	execPath, execErr := os.Executable()
	if execErr != nil {
		return "", false
	}
	binDir := filepath.Dir(execPath)

	currentDir := binDir
	for i := 0; i < 5; i++ {
		vendorRoot := filepath.Join(currentDir, "vendor")
		binaryPath := vendorBinaryPath(vendorRoot, targetTriple, binaryName)
		if fileExists(binaryPath) {
			return binaryPath, true
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	vendorRoot := filepath.Join(binDir, "..", "codex-go-sdk", "vendor")
	binaryPath := vendorBinaryPath(vendorRoot, targetTriple, binaryName)
	if fileExists(binaryPath) {
		return binaryPath, true
	}
	return "", false
}

func findBinaryFromCwd(targetTriple, binaryName string) (string, bool) {
	vendorRoot := filepath.Join("..", "codex-go-sdk", "vendor")
	binaryPath := vendorBinaryPath(vendorRoot, targetTriple, binaryName)
	if fileExists(binaryPath) {
		return binaryPath, true
	}
	return "", false
}

func findBinaryFromHome(targetTriple, binaryName string) (string, bool) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	vendorRoot := filepath.Join(homeDir, ".codex-orchestrator", "vendor")
	binaryPath := vendorBinaryPath(vendorRoot, targetTriple, binaryName)
	if fileExists(binaryPath) {
		return binaryPath, true
	}
	return "", false
}

func findBinaryInPath(binaryName string) (string, bool) {
	path, err := exec.LookPath(binaryName)
	if err != nil {
		return "", false
	}
	return path, true
}

func vendorBinaryPath(vendorRoot, targetTriple, binaryName string) string {
	return filepath.Join(vendorRoot, targetTriple, "codex", binaryName)
}

func fileExists(path string) bool {
	_, statErr := os.Stat(path)
	return statErr == nil
}
