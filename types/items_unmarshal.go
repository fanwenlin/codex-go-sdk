package types

import (
	"encoding/json"
	"errors"
)

// UnknownItem holds raw data for an unrecognized item type.
type UnknownItem struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

// GetType returns the item type discriminator.
func (i UnknownItem) GetType() string {
	return i.Type
}

func unmarshalThreadItem(data []byte) (ThreadItem, error) {
	if len(data) == 0 {
		return nil, errors.New("missing item data")
	}

	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	if meta.Type == "" {
		return nil, errors.New("missing item type")
	}

	item, ok := newThreadItem(meta.Type)
	if !ok {
		return &UnknownItem{
			Type: meta.Type,
			Raw:  append([]byte(nil), data...),
		}, nil
	}
	if err := json.Unmarshal(data, item); err != nil {
		return nil, err
	}
	return item, nil
}

func newThreadItem(itemType string) (ThreadItem, bool) {
	switch itemType {
	case "agentMessage":
		return &AgentMessageItem{}, true
	case "agent_message":
		return &AgentMessageItem{}, true
	case "reasoning":
		return &ReasoningItem{}, true
	case "userMessage":
		return &UserMessageItem{}, true
	case "commandExecution":
		return &CommandExecutionItem{}, true
	case "command_execution":
		return &CommandExecutionItem{}, true
	case "fileChange":
		return &FileChangeItem{}, true
	case "file_change":
		return &FileChangeItem{}, true
	case "mcpToolCall":
		return &McpToolCallItem{}, true
	case "mcp_tool_call":
		return &McpToolCallItem{}, true
	case "webSearch":
		return &WebSearchItem{}, true
	case "web_search":
		return &WebSearchItem{}, true
	case "todoList":
		return &TodoListItem{}, true
	case "todo_list":
		return &TodoListItem{}, true
	case "error":
		return &ErrorItem{}, true
	case "imageView":
		return &ImageViewItem{}, true
	case "image_view":
		return &ImageViewItem{}, true
	case "enteredReviewMode":
		return &EnteredReviewModeItem{}, true
	case "exitedReviewMode":
		return &ExitedReviewModeItem{}, true
	case "compacted":
		return &CompactedItem{}, true
	case "collabToolCall":
		return &CollabToolCallItem{}, true
	default:
		return nil, false
	}
}
