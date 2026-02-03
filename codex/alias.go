package codex

// Re-export all types from the types package.
import (
	"github.com/fanwenlin/codex-go-sdk/types"
)

// Type aliases for convenience.
type (
	// ApprovalMode represents the approval mode for actions.
	ApprovalMode = types.ApprovalMode
	// ApprovalDecision represents the decision for an approval request.
	ApprovalDecision = types.ApprovalDecision
	// TransportMode represents the backend transport.
	TransportMode = types.TransportMode
	// SandboxMode represents the sandbox access mode.
	SandboxMode = types.SandboxMode
	// ModelReasoningEffort represents the reasoning effort level for the model.
	ModelReasoningEffort = types.ModelReasoningEffort
	// WebSearchMode represents the web search mode.
	WebSearchMode = types.WebSearchMode
	// CommandExecutionStatus represents the status of a command execution.
	CommandExecutionStatus = types.CommandExecutionStatus
	// PatchChangeKind indicates the type of file change.
	PatchChangeKind = types.PatchChangeKind
	// PatchApplyStatus represents the status of a file change.
	PatchApplyStatus = types.PatchApplyStatus
	// McpToolCallStatus represents the status of an MCP tool call.
	McpToolCallStatus = types.McpToolCallStatus
)

// Constant values for TransportMode.
const (
	TransportAppServer = types.TransportAppServer
	TransportCLI       = types.TransportCLI
)

// Constant values for ApprovalMode.
const (
	ApprovalModeNever     = types.ApprovalModeNever
	ApprovalModeOnRequest = types.ApprovalModeOnRequest
	ApprovalModeOnFailure = types.ApprovalModeOnFailure
	ApprovalModeUntrusted = types.ApprovalModeUntrusted
)

// Constant values for ApprovalDecision.
const (
	ApprovalDecisionApproved = types.ApprovalDecisionApproved
	ApprovalDecisionRejected = types.ApprovalDecisionRejected
)

// Constant values for SandboxMode.
const (
	SandboxModeReadOnly       = types.SandboxModeReadOnly
	SandboxModeWorkspaceWrite = types.SandboxModeWorkspaceWrite
	SandboxModeFullAccess     = types.SandboxModeFullAccess
)

// Constant values for ModelReasoningEffort.
const (
	ModelReasoningEffortMinimal = types.ModelReasoningEffortMinimal
	ModelReasoningEffortLow     = types.ModelReasoningEffortLow
	ModelReasoningEffortMedium  = types.ModelReasoningEffortMedium
	ModelReasoningEffortHigh    = types.ModelReasoningEffortHigh
	ModelReasoningEffortXHigh   = types.ModelReasoningEffortXHigh
)

// Constant values for WebSearchMode.
const (
	WebSearchModeDisabled = types.WebSearchModeDisabled
	WebSearchModeCached   = types.WebSearchModeCached
	WebSearchModeLive     = types.WebSearchModeLive
)

// Constant values for CommandExecutionStatus.
const (
	CommandExecutionStatusInProgress = types.CommandExecutionStatusInProgress
	CommandExecutionStatusCompleted  = types.CommandExecutionStatusCompleted
	CommandExecutionStatusFailed     = types.CommandExecutionStatusFailed
	CommandExecutionStatusDeclined   = types.CommandExecutionStatusDeclined
)

// Re-export event types.
type (
	ThreadEvent        = types.ThreadEvent
	ThreadStartedEvent = types.ThreadStartedEvent
	TurnStartedEvent   = types.TurnStartedEvent
	Usage              = types.Usage
	TurnCompletedEvent = types.TurnCompletedEvent
	ThreadError        = types.ThreadError
	TurnFailedEvent    = types.TurnFailedEvent
	ItemStartedEvent   = types.ItemStartedEvent
	ItemUpdatedEvent   = types.ItemUpdatedEvent
	ItemCompletedEvent = types.ItemCompletedEvent
	ThreadErrorEvent   = types.ThreadErrorEvent
	RawEvent           = types.RawEvent
)

// Re-export item types.
type (
	ThreadItem            = types.ThreadItem
	CommandExecutionItem  = types.CommandExecutionItem
	FileUpdateChange      = types.FileUpdateChange
	FileChangeItem        = types.FileChangeItem
	McpToolCallItem       = types.McpToolCallItem
	McpToolCallResult     = types.McpToolCallResult
	McpToolCallError      = types.McpToolCallError
	AgentMessageItem      = types.AgentMessageItem
	ReasoningItem         = types.ReasoningItem
	WebSearchItem         = types.WebSearchItem
	TodoItem              = types.TodoItem
	TodoListItem          = types.TodoListItem
	ErrorItem             = types.ErrorItem
	UserMessageItem       = types.UserMessageItem
	ImageViewItem         = types.ImageViewItem
	EnteredReviewModeItem = types.EnteredReviewModeItem
	ExitedReviewModeItem  = types.ExitedReviewModeItem
	CompactedItem         = types.CompactedItem
	CollabToolCallItem    = types.CollabToolCallItem
)

// Re-export option types.
//
//nolint:revive // Keep name for public API compatibility and alignment with the TypeScript SDK.
type CodexOptions = types.CodexOptions

type (
	// ClientInfo identifies the SDK client sending requests.
	ClientInfo = types.ClientInfo
	// ThreadOptions configures thread creation and resume behavior.
	ThreadOptions = types.ThreadOptions
	// TurnOptions configures turn execution behavior.
	TurnOptions = types.TurnOptions
	// ApprovalRequest carries a pending approval request payload.
	ApprovalRequest = types.ApprovalRequest
	// ApprovalHandler handles approval requests from the app server.
	ApprovalHandler = types.ApprovalHandler
)

// Re-export alias types.
type (
	UserInput         = types.UserInput
	Input             = types.Input
	Turn              = types.Turn
	StreamedTurn      = types.StreamedTurn
	RunResult         = types.Turn
	RunStreamedResult = types.StreamedTurn
)

// Helper functions for creating inputs.
//
//nolint:gochecknoglobals // These are intentional function aliases for convenience
var (
	NewTextInput     = types.NewTextInput
	NewImageInput    = types.NewImageInput
	NewImageURLInput = types.NewImageURLInput
	NewSkillInput    = types.NewSkillInput
	NewMentionInput  = types.NewMentionInput
)

// Classes and functions are already exported from other files
// No need to re-export them here as they would create circular references
