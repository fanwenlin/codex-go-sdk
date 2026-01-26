package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fanwenlin/codex-go-sdk/types"
)

// Thread represents a conversation thread with the agent.
type Thread struct {
	exec          Exec
	options       types.CodexOptions
	id            *string
	threadOptions types.ThreadOptions
}

// ID returns the thread ID. Populated after the first turn starts.
func (t *Thread) ID() *string {
	return t.id
}

// newThread creates a new Thread instance.
func newThread(exec Exec, options types.CodexOptions, threadOptions types.ThreadOptions, id *string) *Thread {
	return &Thread{
		exec:          exec,
		options:       options,
		id:            id,
		threadOptions: threadOptions,
	}
}

// RunStreamed provides input to the agent and streams events as they are produced.
func (t *Thread) RunStreamed(input types.Input, turnOptions types.TurnOptions) (*types.StreamedTurn, error) {
	events, err := t.runStreamedInternal(input, turnOptions)
	if err != nil {
		return nil, err
	}
	return &types.StreamedTurn{Events: events}, nil
}

// runStreamedInternal is the internal implementation that generates events.
func (t *Thread) runStreamedInternal(input types.Input, turnOptions types.TurnOptions) (chan types.ThreadEvent, error) {
	// Create output schema file if needed
	schemaFile, err := CreateOutputSchemaFile(turnOptions.OutputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to create output schema file: %w", err)
	}

	// Normalize input
	prompt, images := t.normalizeInput(input)

	// Build context
	ctx := context.Background()
	if turnOptions.Context != nil {
		if c, ok := turnOptions.Context.(context.Context); ok {
			ctx = c
		}
	}

	// Prepare options
	options := t.threadOptions

	// Build arguments
	var threadId *string
	if t.id != nil {
		threadId = t.id
	}

	args := CodexExecArgs{
		Input:                 prompt,
		BaseUrl:               t.options.BaseUrl,
		ApiKey:                t.options.ApiKey,
		ThreadId:              threadId,
		Images:                images,
		Model:                 options.Model,
		SandboxMode:           string(options.SandboxMode),
		WorkingDirectory:      options.WorkingDirectory,
		SkipGitRepoCheck:      options.SkipGitRepoCheck,
		DisableSkills:         options.DisableSkills,
		OutputSchemaFile:      schemaFile.SchemaPath,
		ModelReasoningEffort:  string(options.ModelReasoningEffort),
		Context:               ctx,
		NetworkAccessEnabled:  options.NetworkAccessEnabled,
		WebSearchMode:         string(options.WebSearchMode),
		WebSearchEnabled:      options.WebSearchEnabled,
		ApprovalPolicy:        string(options.ApprovalPolicy),
		AdditionalDirectories: options.AdditionalDirectories,
	}

	events := make(chan types.ThreadEvent)

	go func() {
		defer close(events)
		defer func() {
			if err := schemaFile.Cleanup(); err != nil {
				// Log cleanup error but don't fail
				fmt.Fprintf(os.Stderr, "Warning: failed to cleanup schema file: %v\n", err)
			}
		}()

		// Run the exec command
		resultChan := t.exec.Run(args)

		for result := range resultChan {
			if result.Error != nil {
				// Send error as event and stop
				events <- &types.ThreadErrorEvent{
					Type:    "error",
					Message: result.Error.Error(),
				}
				return
			}

			// Parse the event
			var rawEvent map[string]interface{}
			if err := json.Unmarshal([]byte(result.Line), &rawEvent); err != nil {
				events <- &types.ThreadErrorEvent{
					Type:    "error",
					Message: fmt.Sprintf("failed to parse event: %v", err),
				}
				return
			}

			eventType, ok := rawEvent["type"].(string)
			if !ok {
				events <- &types.ThreadErrorEvent{
					Type:    "error",
					Message: fmt.Sprintf("event missing type field: %s", result.Line),
				}
				return
			}

			var event types.ThreadEvent
			switch eventType {
			case "thread.started":
				threadStarted := &types.ThreadStartedEvent{}
				if err := json.Unmarshal([]byte(result.Line), threadStarted); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal thread.started: %v", err),
					}
					return
				}
				event = threadStarted
				// Set thread ID
				t.id = &threadStarted.ThreadId
			case "turn.started":
				turnStarted := &types.TurnStartedEvent{}
				if err := json.Unmarshal([]byte(result.Line), turnStarted); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal turn.started: %v", err),
					}
					return
				}
				event = turnStarted
			case "turn.completed":
				turnCompleted := &types.TurnCompletedEvent{}
				if err := json.Unmarshal([]byte(result.Line), turnCompleted); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal turn.completed: %v", err),
					}
					return
				}
				event = turnCompleted
			case "turn.failed":
				turnFailed := &types.TurnFailedEvent{}
				if err := json.Unmarshal([]byte(result.Line), turnFailed); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal turn.failed: %v", err),
					}
					return
				}
				event = turnFailed
			case "item.started":
				itemStarted := &types.ItemStartedEvent{}
				if err := json.Unmarshal([]byte(result.Line), itemStarted); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal item.started: %v", err),
					}
					return
				}
				event = itemStarted
			case "item.updated":
				itemUpdated := &types.ItemUpdatedEvent{}
				if err := json.Unmarshal([]byte(result.Line), itemUpdated); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal item.updated: %v", err),
					}
					return
				}
				event = itemUpdated
			case "item.completed":
				itemCompleted := &types.ItemCompletedEvent{}
				if err := json.Unmarshal([]byte(result.Line), itemCompleted); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal item.completed: %v", err),
					}
					return
				}
				event = itemCompleted
			case "error":
				errorEvent := &types.ThreadErrorEvent{}
				if err := json.Unmarshal([]byte(result.Line), errorEvent); err != nil {
					events <- &types.ThreadErrorEvent{
						Type:    "error",
						Message: fmt.Sprintf("failed to unmarshal error: %v", err),
					}
					return
				}
				event = errorEvent
			default:
				events <- &types.ThreadErrorEvent{
					Type:    "error",
					Message: fmt.Sprintf("unknown event type: %s", eventType),
				}
				return
			}

			// Send event to channel
			select {
			case events <- event:
			case <-args.Context.Done():
				return
			}
		}
	}()

	return events, nil
}

// Run provides input to the agent and returns the completed turn.
func (t *Thread) Run(input types.Input, turnOptions types.TurnOptions) (*types.Turn, error) {
	events, err := t.runStreamedInternal(input, turnOptions)
	if err != nil {
		return nil, err
	}

	var items []types.ThreadItem
	var finalResponse string
	var usage *types.Usage
	var turnFailure error

	for event := range events {
		switch e := event.(type) {
		case *types.ItemCompletedEvent:
			// Check if this is an agent message to get the final response
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

	return &types.Turn{
		Items:         items,
		FinalResponse: finalResponse,
		Usage:         usage,
	}, nil
}

// normalizeInput normalizes the input into a prompt string and image paths.
func (t *Thread) normalizeInput(input types.Input) (prompt string, images []string) {
	// Check if input is a string
	if s, ok := input.(string); ok {
		return s, []string{}
	}

	// Otherwise, assume it's a slice of UserInput
	var promptParts []string
	var imagePaths []string

	// Try to parse as []UserInput
	if inputs, ok := input.([]types.UserInput); ok {
		for _, item := range inputs {
			switch item.Type {
			case "text":
				promptParts = append(promptParts, item.Text)
			case "local_image":
				imagePaths = append(imagePaths, item.Path)
			}
		}
	}

	return strings.Join(promptParts, "\n\n"), imagePaths
}
