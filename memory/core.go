package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

// DefaultPersonaLimit is the maximum character length for the persona block.
const DefaultPersonaLimit = 2000

// DefaultHumanLimit is the maximum character length for the human block.
const DefaultHumanLimit = 2000

// CoreConfig configures the core memory tier.
type CoreConfig struct {
	// PersonaLimit is the maximum character count for the persona text block.
	// Defaults to DefaultPersonaLimit (2000).
	PersonaLimit int

	// HumanLimit is the maximum character count for the human text block.
	// Defaults to DefaultHumanLimit (2000).
	HumanLimit int

	// SelfEditable controls whether the agent can modify its own persona
	// and human blocks at runtime.
	SelfEditable bool
}

// Core implements the MemGPT core memory tier. It holds persona and human
// text blocks that are always included in the LLM context window. The agent
// can optionally self-edit these blocks if SelfEditable is true.
//
// Core memory is designed for small, high-value information that the agent
// needs constant access to — its identity (persona) and knowledge about the
// user (human).
type Core struct {
	mu           sync.RWMutex
	persona      string
	human        string
	personaLimit int
	humanLimit   int
	selfEditable bool
}

// NewCore creates a new Core memory with the given configuration. Zero-value
// limits are replaced with defaults.
func NewCore(cfg CoreConfig) *Core {
	pl := cfg.PersonaLimit
	if pl <= 0 {
		pl = DefaultPersonaLimit
	}
	hl := cfg.HumanLimit
	if hl <= 0 {
		hl = DefaultHumanLimit
	}
	return &Core{
		personaLimit: pl,
		humanLimit:   hl,
		selfEditable: cfg.SelfEditable,
	}
}

// GetPersona returns the current persona text.
func (c *Core) GetPersona() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.persona
}

// SetPersona sets the persona text. Returns an error if the text exceeds
// the configured PersonaLimit or if the core is not self-editable and ctx
// indicates an agent call (future use).
func (c *Core) SetPersona(persona string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(persona) > c.personaLimit {
		return fmt.Errorf("memory/core: persona exceeds limit (%d > %d)", len(persona), c.personaLimit)
	}
	c.persona = persona
	return nil
}

// GetHuman returns the current human text.
func (c *Core) GetHuman() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.human
}

// SetHuman sets the human description text. Returns an error if the text
// exceeds the configured HumanLimit.
func (c *Core) SetHuman(human string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(human) > c.humanLimit {
		return fmt.Errorf("memory/core: human exceeds limit (%d > %d)", len(human), c.humanLimit)
	}
	c.human = human
	return nil
}

// IsSelfEditable returns whether the agent can modify core memory blocks.
func (c *Core) IsSelfEditable() bool {
	return c.selfEditable
}

// ToMessages returns the persona and human blocks as system messages suitable
// for inclusion in the LLM context window. Empty blocks are omitted.
func (c *Core) ToMessages() []schema.Message {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var msgs []schema.Message
	if c.persona != "" {
		msgs = append(msgs, schema.NewSystemMessage(fmt.Sprintf("[Persona]\n%s", c.persona)))
	}
	if c.human != "" {
		msgs = append(msgs, schema.NewSystemMessage(fmt.Sprintf("[Human]\n%s", c.human)))
	}
	return msgs
}

// Save implements Memory. Core memory stores the latest input/output pair
// by appending relevant information (this is a simplified implementation;
// a full MemGPT agent would use tool calls to edit core memory blocks).
func (c *Core) Save(ctx context.Context, input, output schema.Message) error {
	// Core memory save is a no-op — the agent edits core memory explicitly
	// via SetPersona/SetHuman tool calls rather than implicit saves.
	return nil
}

// Load implements Memory. Returns the core memory blocks as system messages.
func (c *Core) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return c.ToMessages(), nil
}

// Search implements Memory. Core memory does not support document search;
// it always returns nil.
func (c *Core) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return nil, nil
}

// Clear implements Memory. Resets both persona and human blocks.
func (c *Core) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.persona = ""
	c.human = ""
	return nil
}

func init() {
	Register("core", func(cfg config.ProviderConfig) (Memory, error) {
		coreCfg := CoreConfig{
			PersonaLimit: DefaultPersonaLimit,
			HumanLimit:   DefaultHumanLimit,
		}
		if v, ok := config.GetOption[float64](cfg, "persona_limit"); ok {
			coreCfg.PersonaLimit = int(v)
		}
		if v, ok := config.GetOption[float64](cfg, "human_limit"); ok {
			coreCfg.HumanLimit = int(v)
		}
		if v, ok := config.GetOption[bool](cfg, "self_editable"); ok {
			coreCfg.SelfEditable = v
		}
		return NewCore(coreCfg), nil
	})
}
