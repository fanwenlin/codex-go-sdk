package tests

import (
	"encoding/json"
	"testing"

	"github.com/fanwenlin/codex-go-sdk/codex"
	"github.com/fanwenlin/codex-go-sdk/types"
)

func TestInputNormalization(t *testing.T) {
	// Create a thread to test normalization
	client := codex.NewCodex(types.CodexOptions{})
	_ = client.StartThread(types.ThreadOptions{})

	// We can't directly test private normalizeInput, so we test through Run
	// For this unit test, we'll just verify the types compile

	// Test string input
	input1 := "Hello, world!"
	if input1 == "" {
		t.Error("String input should not be empty")
	}

	// Test UserInput slice
	input2 := []types.UserInput{
		types.NewTextInput("First part"),
		types.NewTextInput("Second part"),
		types.NewImageInput("/path/to/image.png"),
	}

	if len(input2) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(input2))
	}

	if input2[0].Type != "text" || input2[0].Text != "First part" {
		t.Error("First input should be text type")
	}

	if input2[2].Type != "local_image" || input2[2].Path != "/path/to/image.png" {
		t.Error("Third input should be image type")
	}
}

func TestConstants(t *testing.T) {
	// Test that all constants are defined correctly
	if codex.ApprovalModeNever != "never" {
		t.Error("ApprovalModeNever mismatch")
	}
	if codex.ApprovalModeOnRequest != "on-request" {
		t.Error("ApprovalModeOnRequest mismatch")
	}
	if codex.SandboxModeReadOnly != "read-only" {
		t.Error("SandboxModeReadOnly mismatch")
	}
	if codex.ModelReasoningEffortHigh != "high" {
		t.Error("ModelReasoningEffortHigh mismatch")
	}
	if codex.WebSearchModeLive != "live" {
		t.Error("WebSearchModeLive mismatch")
	}
}

func TestTypeAliases(t *testing.T) {
	// Test that type aliases work
	var event types.ThreadEvent = &types.ThreadStartedEvent{
		Type:     "thread.started",
		ThreadId: "test-123",
	}

	if event.GetType() != "thread.started" {
		t.Error("GetType should return thread.started")
	}

	// Test item types
	var item types.ThreadItem = &types.AgentMessageItem{
		ID:   "msg-1",
		Type: "agentMessage",
		Text: "Hello!",
	}

	if item.GetType() != "agentMessage" {
		t.Error("Item GetType should return agentMessage")
	}

	// Test CommandExecutionItem
	exitCode := 0
	aggregatedOutput := "test\n"
	cmdItem := &types.CommandExecutionItem{
		ID:               "cmd-1",
		Type:             "commandExecution",
		Command:          "echo test",
		AggregatedOutput: &aggregatedOutput,
		ExitCode:         &exitCode,
		Status:           types.CommandExecutionStatusCompleted,
	}

	if cmdItem.Status != types.CommandExecutionStatusCompleted {
		t.Error("Command should be completed")
	}
}

func TestTurnCreation(t *testing.T) {
	// Test Turn structure
	usage := &types.Usage{
		InputTokens:       100,
		CachedInputTokens: 20,
		OutputTokens:      50,
	}

	turn := &types.Turn{
		Items: []types.ThreadItem{
			&types.AgentMessageItem{
				ID:   "msg-1",
				Type: "agentMessage",
				Text: "Test response",
			},
		},
		FinalResponse: "Test response",
		Usage:         usage,
	}

	if turn.FinalResponse != "Test response" {
		t.Error("Final response mismatch")
	}

	if turn.Usage.InputTokens != 100 {
		t.Error("Input tokens mismatch")
	}
}

func TestCodexClientCreation(t *testing.T) {
	// Test creating Codex client with options
	client := codex.NewCodex(types.CodexOptions{
		ApiKey:  "test-key",
		BaseUrl: "https://api.example.com",
	})

	if client == nil {
		t.Error("Client should not be nil")
	}

	// Test creating thread
	thread := client.StartThread(types.ThreadOptions{
		Model:       "claude-3-opus",
		SandboxMode: codex.SandboxModeReadOnly,
	})

	if thread == nil {
		t.Error("Thread should not be nil")
	}

	// Thread ID should be nil initially
	if thread.ID() != nil {
		t.Error("Thread ID should be nil before first turn")
	}
}

func TestThreadResumption(t *testing.T) {
	client := codex.NewCodex(types.CodexOptions{})

	// Create thread with ID
	threadID := "existing-thread-123"
	thread := client.ResumeThread(threadID, types.ThreadOptions{})

	if thread == nil {
		t.Error("Resumed thread should not be nil")
	}

	id := thread.ID()
	if id == nil || *id != threadID {
		t.Error("Thread ID should match resumed ID")
	}
}

func TestOutputSchemaFile(t *testing.T) {
	// Test creating output schema file
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"answer": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"answer"},
	}

	schemaFile, err := codex.CreateOutputSchemaFile(schema)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	if schemaFile.SchemaPath == "" {
		t.Error("Schema path should not be empty")
	}

	// Cleanup should work
	err = schemaFile.Cleanup()
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Test nil schema (should return no-op cleanup)
	schemaFile2, err := codex.CreateOutputSchemaFile(nil)
	if err != nil {
		t.Fatalf("Failed to create nil schema file: %v", err)
	}

	if schemaFile2.SchemaPath != "" {
		t.Error("Nil schema should have empty path")
	}

	err = schemaFile2.Cleanup()
	if err != nil {
		t.Errorf("Nil schema cleanup failed: %v", err)
	}
}

func TestEventTypes(t *testing.T) {
	// Test all event types can be created
	events := []types.ThreadEvent{
		&types.ThreadStartedEvent{
			Type:     "thread.started",
			ThreadId: "thread-123",
		},
		&types.TurnStartedEvent{
			Type: "turn.started",
		},
		&types.TurnCompletedEvent{
			Type: "turn.completed",
			Usage: types.Usage{
				InputTokens:       42,
				CachedInputTokens: 12,
				OutputTokens:      5,
			},
		},
		&types.ItemCompletedEvent{
			Type: "item.completed",
			Item: &types.AgentMessageItem{
				ID:   "msg-1",
				Type: "agentMessage",
				Text: "Test",
			},
		},
		&types.ThreadErrorEvent{
			Type:    "error",
			Message: "Test error",
		},
	}

	for _, event := range events {
		if event.GetType() == "" {
			t.Error("Event type should not be empty")
		}
	}
}

func TestItemTypes(t *testing.T) {
	// Test all item types
	items := []types.ThreadItem{
		&types.CommandExecutionItem{
			ID:      "cmd-1",
			Type:    "commandExecution",
			Command: "ls -la",
			Status:  types.CommandExecutionStatusInProgress,
		},
		&types.FileChangeItem{
			ID:   "file-1",
			Type: "fileChange",
			Changes: []types.FileUpdateChange{
				{Path: "test.go", Kind: types.PatchChangeKind{Type: types.PatchChangeKindUpdate}},
			},
			Status: types.PatchApplyStatusCompleted,
		},
		&types.McpToolCallItem{
			ID:        "mcp-1",
			Type:      "mcpToolCall",
			Server:    "test-server",
			Tool:      "test-tool",
			Arguments: map[string]interface{}{"arg": "value"},
			Status:    types.McpToolCallStatusCompleted,
		},
		&types.WebSearchItem{
			ID:    "search-1",
			Type:  "webSearch",
			Query: "test query",
		},
		&types.TodoListItem{
			ID:   "todo-1",
			Type: "todoList",
			Items: []types.TodoItem{
				{Text: "Task 1", Completed: false},
			},
		},
		&types.ErrorItem{
			ID:      "error-1",
			Type:    "error",
			Message: "Test error",
		},
	}

	for _, item := range items {
		if item.GetType() == "" {
			t.Error("Item type should not be empty")
		}
	}
}

func TestItemStartedEvent_FileChangeKindObject(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "fileChange",
			"id": "file-1",
			"changes": [
				{"path": "test.go", "kind": {"type": "update", "move_path": "new_test.go"}}
			],
			"status": "completed"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	if event.GetType() != "item.started" {
		t.Fatalf("expected event type item.started, got %q", event.GetType())
	}

	item, ok := event.Item.(*types.FileChangeItem)
	if !ok {
		t.Fatalf("expected FileChangeItem, got %T", event.Item)
	}
	if len(item.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(item.Changes))
	}
	if item.Changes[0].Kind.Type != types.PatchChangeKindUpdate {
		t.Fatalf("expected kind %q, got %q", types.PatchChangeKindUpdate, item.Changes[0].Kind.Type)
	}
	if item.Changes[0].Kind.MovePath == nil || *item.Changes[0].Kind.MovePath != "new_test.go" {
		t.Fatalf("expected move_path %q, got %v", "new_test.go", item.Changes[0].Kind.MovePath)
	}
}

func TestItemStartedEvent_CommandExecutionSchema(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "commandExecution",
			"id": "cmd-1",
			"command": "ls",
			"aggregatedOutput": "ok",
			"status": "inProgress"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.CommandExecutionItem)
	if !ok {
		t.Fatalf("expected CommandExecutionItem, got %T", event.Item)
	}
	if item.AggregatedOutput == nil || *item.AggregatedOutput != "ok" {
		t.Fatalf("expected aggregated output %q, got %v", "ok", item.AggregatedOutput)
	}
	if string(item.Status) != "inProgress" {
		t.Fatalf("expected status %q, got %q", "inProgress", item.Status)
	}
}

func TestItemCompletedEvent_McpToolCallSchema(t *testing.T) {
	payload := []byte(`{
		"type": "item.completed",
		"item": {
			"type": "mcpToolCall",
			"id": "mcp-1",
			"server": "test-server",
			"tool": "test-tool",
			"arguments": {"arg": "value"},
			"result": {
				"content": "ok",
				"structuredContent": {"a": 1}
			},
			"status": "completed"
		}
	}`)

	var event types.ItemCompletedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.completed failed: %v", err)
	}
	item, ok := event.Item.(*types.McpToolCallItem)
	if !ok {
		t.Fatalf("expected McpToolCallItem, got %T", event.Item)
	}
	if item.Result == nil {
		t.Fatalf("expected result, got nil")
	}
	if item.Result.Content != "ok" {
		t.Fatalf("expected content %q, got %v", "ok", item.Result.Content)
	}
	if item.Result.StructuredContent == nil {
		t.Fatalf("expected structured content, got nil")
	}
}

func TestItemCompletedEvent_FileChangeDeclined(t *testing.T) {
	payload := []byte(`{
		"type": "item.completed",
		"item": {
			"type": "fileChange",
			"id": "file-1",
			"changes": [
				{"path": "test.go", "kind": {"type": "update", "move_path": null}}
			],
			"status": "declined"
		}
	}`)

	var event types.ItemCompletedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.completed failed: %v", err)
	}
	item, ok := event.Item.(*types.FileChangeItem)
	if !ok {
		t.Fatalf("expected FileChangeItem, got %T", event.Item)
	}
	if string(item.Status) != "declined" {
		t.Fatalf("expected status %q, got %q", "declined", item.Status)
	}
	if item.Changes[0].Kind.Type != types.PatchChangeKindUpdate {
		t.Fatalf("expected kind %q, got %q", types.PatchChangeKindUpdate, item.Changes[0].Kind.Type)
	}
	if item.Changes[0].Kind.MovePath != nil {
		t.Fatalf("expected move_path nil, got %v", item.Changes[0].Kind.MovePath)
	}
}

func TestItemCompletedEvent_CommandExecutionDeclined(t *testing.T) {
	payload := []byte(`{
		"type": "item.completed",
		"item": {
			"type": "commandExecution",
			"id": "cmd-2",
			"command": "rm -rf /tmp/nope",
			"aggregatedOutput": "",
			"status": "declined"
		}
	}`)

	var event types.ItemCompletedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.completed failed: %v", err)
	}
	item, ok := event.Item.(*types.CommandExecutionItem)
	if !ok {
		t.Fatalf("expected CommandExecutionItem, got %T", event.Item)
	}
	if item.Status != types.CommandExecutionStatusDeclined {
		t.Fatalf("expected status %q, got %q", types.CommandExecutionStatusDeclined, item.Status)
	}
}

func TestThreadStartedEvent_CamelCaseThreadId(t *testing.T) {
	payload := []byte(`{
		"type": "thread.started",
		"threadId": "thread-123"
	}`)

	var event types.ThreadStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal thread.started failed: %v", err)
	}
	if event.ThreadId != "thread-123" {
		t.Fatalf("expected thread id %q, got %q", "thread-123", event.ThreadId)
	}
}

func TestTurnCompletedEvent_UsageCamelCase(t *testing.T) {
	payload := []byte(`{
		"type": "turn.completed",
		"usage": {
			"inputTokens": 10,
			"cachedInputTokens": 2,
			"outputTokens": 5
		}
	}`)

	var event types.TurnCompletedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal turn.completed failed: %v", err)
	}
	if event.Usage.InputTokens != 10 {
		t.Fatalf("expected input tokens %d, got %d", 10, event.Usage.InputTokens)
	}
	if event.Usage.CachedInputTokens != 2 {
		t.Fatalf("expected cached input tokens %d, got %d", 2, event.Usage.CachedInputTokens)
	}
	if event.Usage.OutputTokens != 5 {
		t.Fatalf("expected output tokens %d, got %d", 5, event.Usage.OutputTokens)
	}
}

func TestItemUpdatedEvent_ReasoningSummaryText(t *testing.T) {
	payload := []byte(`{
		"type": "item.updated",
		"item": {
			"type": "reasoning",
			"id": "reason-1",
			"summary": ["Short summary"]
		}
	}`)

	var event types.ItemUpdatedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.updated failed: %v", err)
	}
	item, ok := event.Item.(*types.ReasoningItem)
	if !ok {
		t.Fatalf("expected ReasoningItem, got %T", event.Item)
	}
	if len(item.Summary) != 1 || item.Summary[0] != "Short summary" {
		t.Fatalf("expected summary %q, got %v", "Short summary", item.Summary)
	}
}

func TestItemStartedEvent_TodoList(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "todoList",
			"id": "todo-1",
			"items": [
				{"text": "Task 1", "completed": false},
				{"text": "Task 2", "completed": true}
			]
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.TodoListItem)
	if !ok {
		t.Fatalf("expected TodoListItem, got %T", event.Item)
	}
	if len(item.Items) != 2 {
		t.Fatalf("expected 2 todo items, got %d", len(item.Items))
	}
	if item.Items[1].Completed != true {
		t.Fatalf("expected second todo completed true, got %v", item.Items[1].Completed)
	}
}

func TestItemStartedEvent_WebSearch(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "webSearch",
			"id": "search-1",
			"query": "hello"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.WebSearchItem)
	if !ok {
		t.Fatalf("expected WebSearchItem, got %T", event.Item)
	}
	if item.Query != "hello" {
		t.Fatalf("expected query %q, got %q", "hello", item.Query)
	}
}

func TestItemStartedEvent_ErrorItem(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "error",
			"id": "error-1",
			"message": "oops"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.ErrorItem)
	if !ok {
		t.Fatalf("expected ErrorItem, got %T", event.Item)
	}
	if item.Message != "oops" {
		t.Fatalf("expected message %q, got %q", "oops", item.Message)
	}
}

func TestItemStartedEvent_UserMessage(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "userMessage",
			"id": "user-1",
			"text": "hi"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.UserMessageItem)
	if !ok {
		t.Fatalf("expected UserMessageItem, got %T", event.Item)
	}
	if item.Text != "hi" {
		t.Fatalf("expected text %q, got %q", "hi", item.Text)
	}
}

func TestItemStartedEvent_ImageView(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "imageView",
			"id": "img-1",
			"url": "https://example.com/image.png"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal item.started failed: %v", err)
	}
	item, ok := event.Item.(*types.ImageViewItem)
	if !ok {
		t.Fatalf("expected ImageViewItem, got %T", event.Item)
	}
	if item.URL != "https://example.com/image.png" {
		t.Fatalf("expected url %q, got %q", "https://example.com/image.png", item.URL)
	}
}

func TestItemStartedEvent_ReviewModeItems(t *testing.T) {
	enterPayload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "enteredReviewMode",
			"id": "enter-1"
		}
	}`)

	var enterEvent types.ItemStartedEvent
	if err := json.Unmarshal(enterPayload, &enterEvent); err != nil {
		t.Fatalf("unmarshal enteredReviewMode failed: %v", err)
	}
	if _, ok := enterEvent.Item.(*types.EnteredReviewModeItem); !ok {
		t.Fatalf("expected EnteredReviewModeItem, got %T", enterEvent.Item)
	}

	exitPayload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "exitedReviewMode",
			"id": "exit-1"
		}
	}`)

	var exitEvent types.ItemStartedEvent
	if err := json.Unmarshal(exitPayload, &exitEvent); err != nil {
		t.Fatalf("unmarshal exitedReviewMode failed: %v", err)
	}
	if _, ok := exitEvent.Item.(*types.ExitedReviewModeItem); !ok {
		t.Fatalf("expected ExitedReviewModeItem, got %T", exitEvent.Item)
	}
}

func TestItemStartedEvent_CompactedItem(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "compacted",
			"id": "compact-1",
			"summary": "summary"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal compacted failed: %v", err)
	}
	item, ok := event.Item.(*types.CompactedItem)
	if !ok {
		t.Fatalf("expected CompactedItem, got %T", event.Item)
	}
	if item.Summary != "summary" {
		t.Fatalf("expected summary %q, got %q", "summary", item.Summary)
	}
}

func TestItemStartedEvent_CollabToolCall(t *testing.T) {
	payload := []byte(`{
		"type": "item.started",
		"item": {
			"type": "collabToolCall",
			"id": "collab-1",
			"tool": "tool",
			"arguments": {"a": 1},
			"status": "inProgress"
		}
	}`)

	var event types.ItemStartedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal collabToolCall failed: %v", err)
	}
	item, ok := event.Item.(*types.CollabToolCallItem)
	if !ok {
		t.Fatalf("expected CollabToolCallItem, got %T", event.Item)
	}
	if item.Tool != "tool" {
		t.Fatalf("expected tool %q, got %q", "tool", item.Tool)
	}
	if item.Status != "inProgress" {
		t.Fatalf("expected status %q, got %q", "inProgress", item.Status)
	}
}
