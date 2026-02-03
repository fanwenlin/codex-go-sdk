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

// ApprovalDecision represents the decision for an approval request.
type ApprovalDecision string

const (
	// ApprovalDecisionApproved means the action is approved.
	ApprovalDecisionApproved ApprovalDecision = "approved"
	// ApprovalDecisionRejected means the action is rejected.
	ApprovalDecisionRejected ApprovalDecision = "rejected"
)

// ApprovalRequest represents an approval request from the app server.
type ApprovalRequest struct {
	ItemID   string
	ItemType string
}

// ApprovalHandler decides how to respond to an approval request.
type ApprovalHandler func(request ApprovalRequest) (ApprovalDecision, error)

// TransportMode represents the backend transport used by the SDK.
type TransportMode string

const (
	// TransportAppServer uses the Codex app server protocol.
	TransportAppServer TransportMode = "app-server"
	// TransportCLI uses the Codex CLI JSONL protocol.
	TransportCLI TransportMode = "cli"
)

// ClientInfo identifies the SDK client to the app server.
type ClientInfo struct {
	Name    string
	Version string
}

// CodexOptions represents options for the Codex client.
type CodexOptions struct {
	// Transport selects the backend transport. Defaults to app-server.
	Transport TransportMode
	// AppServerPathOverride is an optional path to the app server executable.
	// When unset, the SDK will try to locate the codex binary and run it with "app-server".
	AppServerPathOverride string
	// AppServerArgs are optional arguments passed to the app server executable.
	AppServerArgs []string
	// ClientInfo identifies the SDK client to the app server (initialize).
	ClientInfo ClientInfo
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
	// ApprovalHandler handles approval requests when the app server asks for permission.
	ApprovalHandler ApprovalHandler
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
