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
	case "reasoning":
		var item ReasoningItem
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
	case "file_change":
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
	case "web_search":
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
	case "error":
		var item ErrorItem
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
