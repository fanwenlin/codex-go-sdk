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

	switch meta.Type {
	case "agent_message":
		var item AgentMessageItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "agentMessage":
		var item AgentMessageItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "reasoning":
		var item ReasoningItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "userMessage":
		var item UserMessageItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "command_execution":
		var item CommandExecutionItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "commandExecution":
		var item CommandExecutionItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "file_change":
		var item FileChangeItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "fileChange":
		var item FileChangeItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "mcp_tool_call":
		var item McpToolCallItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "mcpToolCall":
		var item McpToolCallItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "web_search":
		var item WebSearchItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "webSearch":
		var item WebSearchItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "todo_list":
		var item TodoListItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "todoList":
		var item TodoListItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "error":
		var item ErrorItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "imageView":
		var item ImageViewItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "enteredReviewMode":
		var item EnteredReviewModeItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "exitedReviewMode":
		var item ExitedReviewModeItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "compacted":
		var item CompactedItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "collabToolCall":
		var item CollabToolCallItem
		if err := json.Unmarshal(data, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "":
		return nil, errors.New("missing item type")
	default:
		return &UnknownItem{
			Type: meta.Type,
			Raw:  append([]byte(nil), data...),
		}, nil
	}
}
