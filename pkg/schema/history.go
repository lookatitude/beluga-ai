package schema

// ChatHistory is an interface for storing and retrieving chat messages.
type ChatHistory interface {
	// AddMessage adds a message to the history.
	AddMessage(message Message) error
	// AddUserMessage adds a user message to the history.
	AddUserMessage(message string) error
	// AddAIMessage adds an AI message to the history.
	AddAIMessage(message string) error
	// Messages returns all messages in the history.
	Messages() ([]Message, error)
	// Clear removes all messages from the history.
	Clear() error
}

// BaseChatHistory provides a basic implementation of the ChatHistory interface.
// It stores messages in memory.
type BaseChatHistory struct {
	messages []Message
}

// NewBaseChatHistory creates a new BaseChatHistory.
func NewBaseChatHistory() *BaseChatHistory {
	return &BaseChatHistory{
		messages: make([]Message, 0),
	}
}

// AddMessage adds a message to the history.
func (h *BaseChatHistory) AddMessage(message Message) error {
	h.messages = append(h.messages, message)
	return nil
}

// AddUserMessage adds a user message to the history.
func (h *BaseChatHistory) AddUserMessage(message string) error {
	return h.AddMessage(NewHumanMessage(message))
}

// AddAIMessage adds an AI message to the history.
func (h *BaseChatHistory) AddAIMessage(message string) error {
	return h.AddMessage(NewAIMessage(message))
}

// Messages returns all messages in the history.
func (h *BaseChatHistory) Messages() ([]Message, error) {
	return h.messages, nil
}

// Clear removes all messages from the history.
func (h *BaseChatHistory) Clear() error {
	h.messages = make([]Message, 0)
	return nil
}

