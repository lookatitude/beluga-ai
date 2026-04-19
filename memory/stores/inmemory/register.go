package inmemory

import (
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/memory"
)

// init registers an "inmemory" Memory provider in the memory registry so that
// a blank import of this package surfaces a curated entry via memory.List().
// The Memory wraps this package's MessageStore using memory.NewRecall, which
// is the canonical adapter from MessageStore to the Memory interface.
func init() {
	memory.Register("inmemory", func(_ config.ProviderConfig) (memory.Memory, error) {
		return memory.NewRecall(NewMessageStore()), nil
	})
}
