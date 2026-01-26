# ChatHistory

**Location:** `ChatHistory` lives in `iface` next to `Message`.

**Required:** `AddMessage(message Message) error`, `Messages() ([]Message, error)`, `Clear() error`.

**Optional:** `AddUserMessage(content string) error`, `AddAIMessage(content string) error` — conveniences; `AddMessage` alone is enough for some implementations.
