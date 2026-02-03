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

	prompt, images := t.normalizeInput(input)
	inputItems := t.normalizeInputItems(input)

	ctx := resolveTurnContext(turnOptions)
	args := t.buildExecArgs(ctx, prompt, inputItems, images, schemaFile.SchemaPath)

	events := make(chan types.ThreadEvent)

	go func() {
		defer close(events)
		defer func() {
			if cleanupErr := schemaFile.Cleanup(); cleanupErr != nil {
				// Log cleanup error but don't fail
				fmt.Fprintf(os.Stderr, "Warning: failed to cleanup schema file: %v\n", cleanupErr)
			}
		}()

		resultChan := t.exec.Run(args)

		for result := range resultChan {
			event, eventErr := t.processExecResult(result)
			if eventErr != nil {
				events <- &types.ThreadErrorEvent{
					Type:    "error",
					Message: eventErr.Error(),
				}
				return
			}
			if event == nil {
				continue
			}
			if threadStarted, ok := event.(*types.ThreadStartedEvent); ok {
				t.id = &threadStarted.ThreadId
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return events, nil
}

func resolveTurnContext(turnOptions types.TurnOptions) context.Context {
	if turnOptions.Context != nil {
		if ctx, ok := turnOptions.Context.(context.Context); ok {
			return ctx
		}
	}
	return context.Background()
}

func (t *Thread) buildExecArgs(
	ctx context.Context,
	prompt string,
	inputItems []types.UserInput,
	images []string,
	schemaPath string,
) CodexExecArgs {
	options := t.threadOptions
	threadID := t.id

	return CodexExecArgs{
		Input:                 prompt,
		InputItems:            inputItems,
		BaseUrl:               t.options.BaseUrl,
		ApiKey:                t.options.ApiKey,
		ThreadId:              threadID,
		Images:                images,
		Model:                 options.Model,
		SandboxMode:           string(options.SandboxMode),
		WorkingDirectory:      options.WorkingDirectory,
		SkipGitRepoCheck:      options.SkipGitRepoCheck,
		DisableSkills:         options.DisableSkills,
		OutputSchemaFile:      schemaPath,
		ModelReasoningEffort:  string(options.ModelReasoningEffort),
		Context:               ctx,
		NetworkAccessEnabled:  options.NetworkAccessEnabled,
		WebSearchMode:         string(options.WebSearchMode),
		WebSearchEnabled:      options.WebSearchEnabled,
		ApprovalPolicy:        string(options.ApprovalPolicy),
		ApprovalHandler:       options.ApprovalHandler,
		AdditionalDirectories: options.AdditionalDirectories,
	}
}

func (t *Thread) processExecResult(result ExecResult) (types.ThreadEvent, error) {
	if result.Error != nil {
		return nil, result.Error
	}
	return parseThreadEvent(result.Line)
}

func parseThreadEvent(line string) (types.ThreadEvent, error) {
	eventType, err := extractEventType(line)
	if err != nil {
		return nil, err
	}
	normalizedType := strings.ReplaceAll(eventType, "/", ".")
	if factory, ok := threadEventFactory(normalizedType); ok {
		event := factory()
		if unmarshalErr := json.Unmarshal([]byte(line), event); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", normalizedType, unmarshalErr)
		}
		return event, nil
	}
	return &types.RawEvent{
		Type: eventType,
		Raw:  json.RawMessage(line),
	}, nil
}

func extractEventType(line string) (string, error) {
	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(line), &meta); err != nil {
		return "", fmt.Errorf("failed to parse event: %w", err)
	}
	if meta.Type == "" {
		return "", fmt.Errorf("event missing type field: %s", line)
	}
	return meta.Type, nil
}

func threadEventFactory(eventType string) (func() types.ThreadEvent, bool) {
	factories := map[string]func() types.ThreadEvent{
		"thread.started": func() types.ThreadEvent { return &types.ThreadStartedEvent{} },
		"turn.started":   func() types.ThreadEvent { return &types.TurnStartedEvent{} },
		"turn.completed": func() types.ThreadEvent { return &types.TurnCompletedEvent{} },
		"turn.failed":    func() types.ThreadEvent { return &types.TurnFailedEvent{} },
		"item.started":   func() types.ThreadEvent { return &types.ItemStartedEvent{} },
		"item.updated":   func() types.ThreadEvent { return &types.ItemUpdatedEvent{} },
		"item.completed": func() types.ThreadEvent { return &types.ItemCompletedEvent{} },
		"error":          func() types.ThreadEvent { return &types.ThreadErrorEvent{} },
	}
	factory, ok := factories[eventType]
	return factory, ok
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

// normalizeInputItems preserves structured input items when available.
func (t *Thread) normalizeInputItems(input types.Input) []types.UserInput {
	if s, ok := input.(string); ok {
		return []types.UserInput{types.NewTextInput(s)}
	}
	if inputs, ok := input.([]types.UserInput); ok {
		return inputs
	}
	return nil
}
