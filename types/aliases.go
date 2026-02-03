package types

// UserInput represents an input to send to the agent.
// It can be text, a local image, a remote image, a skill, or a mention.
type UserInput struct {
	// Type is the input type: "text", "local_image", "image", "skill", or "mention"
	Type string `json:"type"`
	// Text is the text content (when Type is "text")
	Text string `json:"text,omitempty"`
	// Path is the local image path (when Type is "local_image")
	Path string `json:"path,omitempty"`
	// URL is the remote image URL (when Type is "image")
	URL string `json:"url,omitempty"`
	// Name is the identifier for skill or mention inputs
	Name string `json:"name,omitempty"`
}

// NewTextInput creates a new text input.
func NewTextInput(text string) UserInput {
	return UserInput{
		Type: "text",
		Text: text,
	}
}

// NewImageInput creates a new local image input.
func NewImageInput(path string) UserInput {
	return UserInput{
		Type: "local_image",
		Path: path,
	}
}

// NewImageURLInput creates a new remote image input.
func NewImageURLInput(url string) UserInput {
	return UserInput{
		Type: "image",
		URL:  url,
	}
}

// NewSkillInput creates a new skill input.
func NewSkillInput(name string) UserInput {
	return UserInput{
		Type: "skill",
		Name: name,
	}
}

// NewMentionInput creates a new mention input.
func NewMentionInput(text string) UserInput {
	return UserInput{
		Type: "mention",
		Text: text,
	}
}

// Input represents input to the agent.
// It can be either a string or a slice of UserInput.
type Input interface{}

// Turn represents a completed turn.
type Turn struct {
	// Items are the thread items generated during the turn
	Items []ThreadItem `json:"items"`
	// FinalResponse is the final response text from the agent
	FinalResponse string `json:"finalResponse"`
	// Usage is the token usage for the turn
	Usage *Usage `json:"usage"`
}

// StreamedTurn represents the result of a streamed turn.
type StreamedTurn struct {
	// Events is a channel of thread events
	Events chan ThreadEvent
}

// RunResult is an alias for Turn.
type RunResult = Turn

// RunStreamedResult is an alias for StreamedTurn.
type RunStreamedResult = StreamedTurn
