package tests

import (
	"strings"
	"testing"

	"github.com/fanwenlin/codex-go-sdk/codex"
	"github.com/fanwenlin/codex-go-sdk/types"
)

// TestCodexExecArgsGeneration tests that command arguments are properly generated
func TestCodexExecArgsGeneration(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{
		Model:          "claude-test",
		SandboxMode:    codex.SandboxModeWorkspaceWrite,
		ApprovalPolicy: codex.ApprovalModeOnRequest,
	})

	_, _ = thread.Run("test input", types.TurnOptions{})

	args := mockExec.GetArgs()
	if len(args) == 0 {
		t.Fatal("No args captured")
	}

	if model, found := FindFlag(args[0], "--model"); !found || model != "claude-test" {
		t.Errorf("Model not found or incorrect: %s", model)
	}

	if sandbox, found := FindFlag(args[0], "--sandbox"); !found || sandbox != "workspace-write" {
		t.Errorf("Sandbox not found or incorrect: %s", sandbox)
	}
}

func TestWebSearchTrue(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{
		WebSearchEnabled: boolPtr(true),
	})

	_, _ = thread.Run("test", types.TurnOptions{})

	args := mockExec.GetArgs()
	if len(args) == 0 {
		t.Fatal("No args captured")
	}

	found := false
	for _, arg := range args[0] {
		if strings.Contains(arg, `web_search="live"`) {
			found = true
			break
		}
	}
	if !found {
		t.Error(`Expected web_search="live" not found`)
	}
}

func TestImagesForwarding(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{})

	input := []types.UserInput{
		types.NewTextInput("Describe images"),
		types.NewImageInput("/path/image1.png"),
		types.NewImageInput("/path/image2.jpg"),
	}

	_, _ = thread.Run(input, types.TurnOptions{})

	args := mockExec.GetArgs()
	if len(args) == 0 {
		t.Fatal("No args captured")
	}

	images := FindAllFlags(args[0], "--image")
	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d: %v", len(images), images)
	}
}

func TestAdditionalDirectories(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{
		AdditionalDirectories: []string{"/path/one", "/path/two"},
	})

	_, _ = thread.Run("test", types.TurnOptions{})

	args := mockExec.GetArgs()
	if len(args) == 0 {
		t.Fatal("No args captured")
	}

	dirs := FindAllFlags(args[0], "--add-dir")
	if len(dirs) != 2 {
		t.Errorf("Expected 2 --add-dir flags, got %d: %v", len(dirs), dirs)
	}
}

func TestOutputSchemaForwarding(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"answer": map[string]interface{}{"type": "string"},
		},
		"required": []string{"answer"},
	}

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{})

	_, _ = thread.Run("test", types.TurnOptions{OutputSchema: schema})

	args := mockExec.GetArgs()
	if len(args) == 0 {
		t.Fatal("No args captured")
	}

	if _, found := FindFlag(args[0], "--output-schema"); !found {
		t.Error("Output schema flag not found")
	}
}

func TestThreadIDSettingFromEvent(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test-thread-123"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{})

	streamed, _ := thread.RunStreamed("test", types.TurnOptions{})
	drainEvents(streamed.Events)

	threadID := thread.ID()
	if threadID == nil || *threadID != "test-thread-123" {
		t.Errorf("Thread ID not set correctly: %v", threadID)
	}
}

func TestRunStreamedEvents(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{
		`{"type":"thread.started","threadId":"test"}`,
		`{"type":"turn.started"}`,
		`{"type":"turn.completed","usage":{"inputTokens":10,"cachedInputTokens":2,"outputTokens":5}}`,
	})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{})

	streamed, err := thread.RunStreamed("test", types.TurnOptions{})
	if err != nil {
		t.Fatalf("RunStreamed failed: %v", err)
	}

	eventCount := 0
	for range streamed.Events {
		eventCount++
	}

	if eventCount != 3 {
		t.Errorf("Expected 3 events, got %d", eventCount)
	}
}

func TestItemCompletedParsing(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{
		`{"type":"thread.started","threadId":"test"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"msg-1","type":"agentMessage","text":"Hello"}}`,
		`{"type":"turn.completed","usage":{"inputTokens":1,"cachedInputTokens":0,"outputTokens":1}}`,
	})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{})
	thread := client.StartThread(types.ThreadOptions{})

	turn, err := thread.Run("test", types.TurnOptions{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if turn.FinalResponse != "Hello" {
		t.Errorf("Final response mismatch: %s", turn.FinalResponse)
	}

	if len(turn.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(turn.Items))
	}

	switch item := turn.Items[0].(type) {
	case *types.AgentMessageItem:
		if item.Text != "Hello" {
			t.Errorf("Agent message text mismatch: %s", item.Text)
		}
	case types.AgentMessageItem:
		if item.Text != "Hello" {
			t.Errorf("Agent message text mismatch: %s", item.Text)
		}
	default:
		t.Fatalf("Unexpected item type: %T", turn.Items[0])
	}
}

func TestEnvPassedToExec(t *testing.T) {
	mockExec := NewMockExec()
	mockExec.SetEvents([]string{`{"type":"thread.started","threadId":"test"}`})

	client := codex.NewCodexWithExec(mockExec, types.CodexOptions{
		ApiKey:  "test-key-123",
		BaseUrl: "https://api.example.com",
	})

	thread := client.StartThread(types.ThreadOptions{})
	_, _ = thread.Run("test", types.TurnOptions{})

	envs := mockExec.GetEnvs()
	if len(envs) == 0 {
		t.Fatal("No envs captured")
	}

	if envs[0]["CODEX_API_KEY"] != "test-key-123" {
		t.Errorf("API key not in env: %v", envs[0])
	}

	if envs[0]["OPENAI_BASE_URL"] != "https://api.example.com" {
		t.Errorf("Base URL not in env: %v", envs[0])
	}

	if envs[0]["CODEX_INTERNAL_ORIGINATOR_OVERRIDE"] != "codex_sdk_go" {
		t.Error("Originator not set correctly")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func drainEvents(events <-chan types.ThreadEvent) {
	for range events {
		// Drain
	}
}
