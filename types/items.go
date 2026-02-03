package types

// CommandExecutionStatus represents the status of a command execution.
type CommandExecutionStatus string

const (
	CommandExecutionStatusInProgress CommandExecutionStatus = "inProgress"
	CommandExecutionStatusCompleted  CommandExecutionStatus = "completed"
	CommandExecutionStatusFailed     CommandExecutionStatus = "failed"
	CommandExecutionStatusDeclined   CommandExecutionStatus = "declined"
)

// CommandExecutionItem represents a command executed by the agent.
type CommandExecutionItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Command is the command line executed by the agent
	Command string `json:"command"`
	// AggregatedOutput is stdout and stderr captured while the command was running
	AggregatedOutput *string `json:"aggregatedOutput,omitempty"`
	// ExitCode is set when the command exits; omitted while still running
	ExitCode *int `json:"exitCode,omitempty"`
	// Status is the current status of the command execution
	Status CommandExecutionStatus `json:"status"`
}

// GetType returns the item type discriminator.
func (i CommandExecutionItem) GetType() string {
	return i.Type
}

// PatchChangeKindType indicates the type of the file change.
type PatchChangeKindType string

const (
	PatchChangeKindAdd    PatchChangeKindType = "add"
	PatchChangeKindDelete PatchChangeKindType = "delete"
	PatchChangeKindUpdate PatchChangeKindType = "update"
)

// PatchChangeKind describes the change, including update move_path when present.
type PatchChangeKind struct {
	Type     PatchChangeKindType `json:"type"`
	MovePath *string             `json:"move_path,omitempty"`
}

// FileUpdateChange represents a set of file changes by the agent.
type FileUpdateChange struct {
	Path string          `json:"path"`
	Kind PatchChangeKind `json:"kind"`
}

// PatchApplyStatus represents the status of a file change.
type PatchApplyStatus string

const (
	PatchApplyStatusInProgress PatchApplyStatus = "inProgress"
	PatchApplyStatusCompleted  PatchApplyStatus = "completed"
	PatchApplyStatusFailed     PatchApplyStatus = "failed"
	PatchApplyStatusDeclined   PatchApplyStatus = "declined"
)

// FileChangeItem represents a set of file changes by the agent.
type FileChangeItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Changes are individual file changes that comprise the patch
	Changes []FileUpdateChange `json:"changes"`
	// Status indicates whether the patch ultimately succeeded or failed
	Status PatchApplyStatus `json:"status"`
	// Output contains streamed output (when available)
	Output string `json:"output,omitempty"`
}

// GetType returns the item type discriminator.
func (i FileChangeItem) GetType() string {
	return i.Type
}

// McpToolCallStatus represents the status of an MCP tool call.
type McpToolCallStatus string

const (
	McpToolCallStatusInProgress McpToolCallStatus = "inProgress"
	McpToolCallStatusCompleted  McpToolCallStatus = "completed"
	McpToolCallStatusFailed     McpToolCallStatus = "failed"
)

// McpToolCallItem represents a call to an MCP tool.
type McpToolCallItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Server is the name of the MCP server handling the request
	Server string `json:"server"`
	// Tool is the tool invoked on the MCP server
	Tool string `json:"tool"`
	// Arguments forwarded to the tool invocation
	Arguments interface{} `json:"arguments"`
	// Result payload returned by the MCP server for successful calls
	Result *McpToolCallResult `json:"result,omitempty"`
	// Error message reported for failed calls
	Error *McpToolCallError `json:"error,omitempty"`
	// Status is the current status of the tool invocation
	Status McpToolCallStatus `json:"status"`
}

// McpToolCallResult contains the result payload for successful MCP tool calls.
type McpToolCallResult struct {
	Content           interface{} `json:"content"`
	StructuredContent interface{} `json:"structuredContent"`
}

// McpToolCallError contains error information for failed MCP tool calls.
type McpToolCallError struct {
	Message string `json:"message"`
}

// GetType returns the item type discriminator.
func (i McpToolCallItem) GetType() string {
	return i.Type
}

// AgentMessageItem represents a response from the agent.
type AgentMessageItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Text is either natural-language text or JSON when structured output is requested
	Text string `json:"text"`
}

// GetType returns the item type discriminator.
func (i AgentMessageItem) GetType() string {
	return i.Type
}

// ReasoningItem represents the agent's reasoning summary.
type ReasoningItem struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Summary []string `json:"summary"`
}

// GetType returns the item type discriminator.
func (i ReasoningItem) GetType() string {
	return i.Type
}

// WebSearchItem captures a web search request.
type WebSearchItem struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Query string `json:"query"`
}

// GetType returns the item type discriminator.
func (i WebSearchItem) GetType() string {
	return i.Type
}

// TodoItem represents an item in the agent's to-do list.
type TodoItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// TodoListItem tracks the agent's running to-do list.
type TodoListItem struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Items []TodoItem `json:"items"`
}

// GetType returns the item type discriminator.
func (i TodoListItem) GetType() string {
	return i.Type
}

// ErrorItem describes a non-fatal error surfaced as an item.
type ErrorItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

// GetType returns the item type discriminator.
func (i ErrorItem) GetType() string {
	return i.Type
}

// ThreadItem is the union type for all thread items.
// All item types must implement this interface.
type ThreadItem interface {
	GetType() string
}

// UserMessageItem represents a user message in the thread.
type UserMessageItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// GetType returns the item type discriminator.
func (i UserMessageItem) GetType() string {
	return i.Type
}

// ImageViewItem represents an image preview item.
type ImageViewItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
}

// GetType returns the item type discriminator.
func (i ImageViewItem) GetType() string {
	return i.Type
}

// EnteredReviewModeItem represents entering review mode.
type EnteredReviewModeItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// GetType returns the item type discriminator.
func (i EnteredReviewModeItem) GetType() string {
	return i.Type
}

// ExitedReviewModeItem represents exiting review mode.
type ExitedReviewModeItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// GetType returns the item type discriminator.
func (i ExitedReviewModeItem) GetType() string {
	return i.Type
}

// CompactedItem represents a compacted summary of the thread.
type CompactedItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Summary string `json:"summary,omitempty"`
}

// GetType returns the item type discriminator.
func (i CompactedItem) GetType() string {
	return i.Type
}

// CollabToolCallItem represents a collaborative tool call.
type CollabToolCallItem struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Tool      string            `json:"tool,omitempty"`
	Arguments interface{}       `json:"arguments,omitempty"`
	Result    interface{}       `json:"result,omitempty"`
	Error     *McpToolCallError `json:"error,omitempty"`
	Status    string            `json:"status,omitempty"`
}

// GetType returns the item type discriminator.
func (i CollabToolCallItem) GetType() string {
	return i.Type
}
