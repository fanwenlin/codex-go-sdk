package types

import "io"

// ApprovalMode represents the approval mode for actions.
type ApprovalMode string

const (
	ApprovalModeNever     ApprovalMode = "never"
	ApprovalModeOnRequest ApprovalMode = "on-request"
	ApprovalModeOnFailure ApprovalMode = "on-failure"
	ApprovalModeUntrusted ApprovalMode = "untrusted"
)

// SandboxMode represents the sandbox access mode.
type SandboxMode string

const (
	SandboxModeReadOnly       SandboxMode = "read-only"
	SandboxModeWorkspaceWrite SandboxMode = "workspace-write"
	SandboxModeFullAccess     SandboxMode = "danger-full-access"
)

// ModelReasoningEffort represents the reasoning effort level for the model.
type ModelReasoningEffort string

const (
	ModelReasoningEffortMinimal ModelReasoningEffort = "minimal"
	ModelReasoningEffortLow     ModelReasoningEffort = "low"
	ModelReasoningEffortMedium  ModelReasoningEffort = "medium"
	ModelReasoningEffortHigh    ModelReasoningEffort = "high"
	ModelReasoningEffortXHigh   ModelReasoningEffort = "xhigh"
)

// WebSearchMode represents the web search mode.
type WebSearchMode string

const (
	WebSearchModeDisabled WebSearchMode = "disabled"
	WebSearchModeCached   WebSearchMode = "cached"
	WebSearchModeLive     WebSearchMode = "live"
)

// CodexOptions represents options for the Codex client.
type CodexOptions struct {
	// CodexPathOverride is an optional path to the codex binary
	CodexPathOverride string
	// BaseUrl is the base URL for the API
	BaseUrl string
	// ApiKey is the API key for authentication
	ApiKey string
	// Env is environment variables passed to the Codex CLI process.
	// When provided, the SDK will not inherit variables from the environment.
	Env map[string]string
	// Verbose enables debug logging for Codex CLI execution.
	Verbose bool
	// VerboseWriter is the output for debug logs when Verbose is enabled.
	VerboseWriter io.Writer
}

// ThreadOptions represents options for a thread.
type ThreadOptions struct {
	// Model is the model to use
	Model string
	// SandboxMode is the sandbox access mode
	SandboxMode SandboxMode
	// WorkingDirectory is the working directory for the thread
	WorkingDirectory string
	// SkipGitRepoCheck skips the git repository check
	SkipGitRepoCheck bool
	// DisableSkills disables the Codex CLI skills feature.
	DisableSkills bool
	// ModelReasoningEffort is the reasoning effort for the model
	ModelReasoningEffort ModelReasoningEffort
	// NetworkAccessEnabled enables network access
	NetworkAccessEnabled bool
	// WebSearchMode is the web search mode
	WebSearchMode WebSearchMode
	// WebSearchEnabled is a legacy flag for web search
	WebSearchEnabled *bool
	// ApprovalPolicy is the approval mode
	ApprovalPolicy ApprovalMode
	// AdditionalDirectories are additional directories to include
	AdditionalDirectories []string
}

// TurnOptions represents options for a turn.
type TurnOptions struct {
	// OutputSchema is a JSON schema describing the expected agent output
	OutputSchema interface{}
	// Context is a context.Context for cancellation (replaces AbortSignal from TypeScript)
	Context interface{}
}
