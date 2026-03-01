package types

import (
	"encoding/json"
	"errors"
)

// ThreadEvent is the union type for all thread events.
// In Go, we implement this using an interface that all event types implement.
type ThreadEvent interface {
	// Type returns the event type discriminator
	GetType() string
}

// ThreadStartedEvent is emitted when a new thread is started as the first event.
type ThreadStartedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// ThreadId is the identifier of the new thread
	ThreadId string `json:"threadId"`
}

// GetType returns the event type discriminator
func (e ThreadStartedEvent) GetType() string {
	return e.Type
}

// TurnStartedEvent is emitted when a turn is started by sending a new prompt to the model.
type TurnStartedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
}

// GetType returns the event type discriminator
func (e TurnStartedEvent) GetType() string {
	return e.Type
}

// Usage describes the usage of tokens during a turn.
type Usage struct {
	// InputTokens is the number of input tokens used during the turn
	InputTokens int `json:"inputTokens"`
	// CachedInputTokens is the number of cached input tokens used during the turn
	CachedInputTokens int `json:"cachedInputTokens"`
	// OutputTokens is the number of output tokens used during the turn
	OutputTokens int `json:"outputTokens"`
}

// TurnCompletedEvent is emitted when a turn is completed.
type TurnCompletedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Usage is the token usage for the turn
	Usage Usage `json:"usage"`
}

// GetType returns the event type discriminator
func (e TurnCompletedEvent) GetType() string {
	return e.Type
}

// ThreadError represents a fatal error emitted by the stream.
type ThreadError struct {
	Message string `json:"message"`
}

// TurnFailedEvent indicates that a turn failed with an error.
type TurnFailedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Error is the error that occurred
	Error ThreadError `json:"error"`
}

// GetType returns the event type discriminator
func (e TurnFailedEvent) GetType() string {
	return e.Type
}

// ItemStartedEvent is emitted when a new item is added to the thread.
type ItemStartedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Item is the thread item that was started
	Item ThreadItem `json:"item"`
}

// UnmarshalJSON parses the item payload into a concrete ThreadItem.
func (e *ItemStartedEvent) UnmarshalJSON(data []byte) error {
	eventType, item, err := unmarshalItemEvent(data)
	if err != nil {
		return err
	}
	e.Type = eventType
	e.Item = item
	return nil
}

// GetType returns the event type discriminator
func (e ItemStartedEvent) GetType() string {
	return e.Type
}

// ItemUpdatedEvent is emitted when an item is updated.
type ItemUpdatedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Item is the updated thread item
	Item ThreadItem `json:"item"`
}

// UnmarshalJSON parses the item payload into a concrete ThreadItem.
func (e *ItemUpdatedEvent) UnmarshalJSON(data []byte) error {
	eventType, item, err := unmarshalItemEvent(data)
	if err != nil {
		return err
	}
	e.Type = eventType
	e.Item = item
	return nil
}

// GetType returns the event type discriminator
func (e ItemUpdatedEvent) GetType() string {
	return e.Type
}

// ItemCompletedEvent signals that an item has reached a terminal state.
type ItemCompletedEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Item is the completed thread item
	Item ThreadItem `json:"item"`
}

// UnmarshalJSON parses the item payload into a concrete ThreadItem.
func (e *ItemCompletedEvent) UnmarshalJSON(data []byte) error {
	eventType, item, err := unmarshalItemEvent(data)
	if err != nil {
		return err
	}
	e.Type = eventType
	e.Item = item
	return nil
}

// GetType returns the event type discriminator
func (e ItemCompletedEvent) GetType() string {
	return e.Type
}

// ThreadErrorEvent represents an unrecoverable error emitted by the event stream.
type ThreadErrorEvent struct {
	// Type is the event type discriminator
	Type string `json:"type"`
	// Message is the error message
	Message string `json:"message"`
}

// UnmarshalJSON supports both legacy and current error payload shapes.
// Some transports emit {"type":"error","message":"..."} while others emit
// {"type":"error","error":{"message":"..."}}.
func (e *ThreadErrorEvent) UnmarshalJSON(data []byte) error {
	var payload struct {
		Type    string       `json:"type"`
		Message string       `json:"message"`
		Error   *ThreadError `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	e.Type = payload.Type
	e.Message = payload.Message
	if e.Message == "" && payload.Error != nil {
		e.Message = payload.Error.Message
	}
	return nil
}

// GetType returns the event type discriminator
func (e ThreadErrorEvent) GetType() string {
	return e.Type
}

// RawEvent preserves unrecognized events from the backend.
type RawEvent struct {
	// Type is the event type discriminator.
	Type string `json:"type"`
	// Raw is the original JSON payload.
	Raw json.RawMessage `json:"raw"`
}

// GetType returns the event type discriminator.
func (e RawEvent) GetType() string {
	return e.Type
}

type itemEventEnvelope struct {
	Type string          `json:"type"`
	Item json.RawMessage `json:"item"`
}

func unmarshalItemEvent(data []byte) (string, ThreadItem, error) {
	var envelope itemEventEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", nil, err
	}
	if len(envelope.Item) == 0 {
		return envelope.Type, nil, errors.New("missing item payload")
	}
	item, err := unmarshalThreadItem(envelope.Item)
	if err != nil {
		return envelope.Type, nil, err
	}
	return envelope.Type, item, nil
}
