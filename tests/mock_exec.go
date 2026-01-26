package tests

import (
	"fmt"
	"sync"

	"github.com/fanwenlin/codex-go-sdk/codex"
	"github.com/fanwenlin/codex-go-sdk/types"
)

// MockExec is a mock implementation of Exec for testing
type MockExec struct {
	Events []string
	Args   [][]string
	Envs   []map[string]string
	mu     sync.Mutex
}

// NewMockExec creates a new mock exec
func NewMockExec() *MockExec {
	return &MockExec{
		Events: []string{},
		Args:   [][]string{},
		Envs:   []map[string]string{},
	}
}

// Run implements the Exec interface
func (m *MockExec) Run(args codex.CodexExecArgs) <-chan codex.ExecResult {
	output := make(chan codex.ExecResult)

	// Record the arguments
	m.mu.Lock()
	m.Args = append(m.Args, buildCommandArgs(args))
	m.Envs = append(m.Envs, buildEnvMap(args))
	m.mu.Unlock()

	go func() {
		defer close(output)

		for _, event := range m.Events {
			output <- codex.ExecResult{Line: event}
		}
	}()

	return output
}

// SetEvents sets the events to return
func (m *MockExec) SetEvents(events []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = events
}

// GetArgs returns captured arguments
func (m *MockExec) GetArgs() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Args
}

// GetEnvs returns captured environments
func (m *MockExec) GetEnvs() []map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Envs
}

// buildCommandArgs builds the command args as exec would
func buildCommandArgs(args codex.CodexExecArgs) []string {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "exec", "--experimental-json")

	if args.Model != "" {
		cmdArgs = append(cmdArgs, "--model", args.Model)
	}
	if args.SandboxMode != "" {
		cmdArgs = append(cmdArgs, "--sandbox", args.SandboxMode)
	}
	if args.WorkingDirectory != "" {
		cmdArgs = append(cmdArgs, "--cd", args.WorkingDirectory)
	}
	for _, dir := range args.AdditionalDirectories {
		cmdArgs = append(cmdArgs, "--add-dir", dir)
	}
	if args.SkipGitRepoCheck {
		cmdArgs = append(cmdArgs, "--skip-git-repo-check")
	}
	if args.OutputSchemaFile != "" {
		cmdArgs = append(cmdArgs, "--output-schema", args.OutputSchemaFile)
	}
	if args.ModelReasoningEffort != "" {
		cmdArgs = append(cmdArgs, "--config", fmt.Sprintf(`model_reasoning_effort="%s"`, args.ModelReasoningEffort))
	}
	if args.NetworkAccessEnabled {
		cmdArgs = append(cmdArgs, "--config", fmt.Sprintf("sandbox_workspace_write.network_access=%t", args.NetworkAccessEnabled))
	}
	if args.WebSearchMode != "" {
		cmdArgs = append(cmdArgs, "--config", fmt.Sprintf(`web_search="%s"`, args.WebSearchMode))
	} else if args.WebSearchEnabled != nil {
		if *args.WebSearchEnabled {
			cmdArgs = append(cmdArgs, "--config", `web_search="live"`)
		} else {
			cmdArgs = append(cmdArgs, "--config", `web_search="disabled"`)
		}
	}
	if args.ApprovalPolicy != "" {
		cmdArgs = append(cmdArgs, "--config", fmt.Sprintf(`approval_policy="%s"`, args.ApprovalPolicy))
	}
	for _, image := range args.Images {
		cmdArgs = append(cmdArgs, "--image", image)
	}
	if args.ThreadId != nil {
		cmdArgs = append(cmdArgs, "resume", *args.ThreadId)
	}

	return cmdArgs
}

// buildEnvMap builds the environment map
func buildEnvMap(args codex.CodexExecArgs) map[string]string {
	env := make(map[string]string)
	env["CODEX_INTERNAL_ORIGINATOR_OVERRIDE"] = "codex_sdk_go"
	if args.BaseUrl != "" {
		env["OPENAI_BASE_URL"] = args.BaseUrl
	}
	if args.ApiKey != "" {
		env["CODEX_API_KEY"] = args.ApiKey
	}
	return env
}

// FindFlag finds a flag value in args
func FindFlag(args []string, flag string) (string, bool) {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// FindAllFlags finds all values for a repeated flag
func FindAllFlags(args []string, flag string) []string {
	var values []string
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			values = append(values, args[i+1])
		}
	}
	return values
}

// BuildMockEvents builds a mock event stream for testing
func BuildMockEvents(responseID, messageID, responseText string) []string {
	return []string{
		fmt.Sprintf(`{"type":"thread.started","thread_id":"thread_%s"}`, responseID),
		`{"type":"turn.started"}`,
		fmt.Sprintf(`{"type":"item.completed","item":{"id":"msg_%s","type":"agent_message","text":%q}}`, messageID, responseText),
		fmt.Sprintf(`{"type":"turn.completed","usage":{"input_tokens":42,"cached_input_tokens":12,"output_tokens":5}}`),
	}
}

// BuildMockEventsWithFailure builds a mock event stream that fails
func BuildMockEventsWithFailure(errorMessage string) []string {
	return []string{
		`{"type":"thread.started","thread_id":"thread_fail"}`,
		`{"type":"turn.started"}`,
		fmt.Sprintf(`{"type":"turn.failed","error":{"message":"%s"}}`, errorMessage),
	}
}

// NewTestCodex creates a new Codex client with a mock exec for testing
func NewTestCodex(mockExec *MockExec) *codex.Codex {
	return codex.NewCodexWithExec(mockExec, types.CodexOptions{})
}

// NewTestThread creates a new Thread with a mock exec for testing
func NewTestThread(mockExec *MockExec) *codex.Thread {
	client := NewTestCodex(mockExec)
	return client.StartThread(types.ThreadOptions{})
}
