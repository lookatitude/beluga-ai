package factory

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

var (
	globalCreators   = make(map[string]func(config map[string]interface{}) (vectorstores.VectorStore, error))
	globalCreatorsMu sync.RWMutex
)

// Register globally adds a new vector store creator function.
// This function is intended to be called from the init() function of vector store provider packages.
func Register(name string, creator func(config map[string]interface{}) (vectorstores.VectorStore, error)) {
	globalCreatorsMu.Lock()
	defer globalCreatorsMu.Unlock()
	if _, dup := globalCreators[name]; dup {
		panic(fmt.Sprintf("Vector store creator for 	%s	 already registered", name))
	}
	globalCreators[name] = creator
}

// VectorStoreFactory is responsible for creating vector store instances.
// It allows for easy addition of new vector store types.
type VectorStoreFactory struct {
	// creators holds a snapshot of the globally registered creators at the time of factory instantiation.
	creators map[string]func(config map[string]interface{}) (vectorstores.VectorStore, error)
}

// NewVectorStoreFactory creates a new instance of VectorStoreFactory.
// It initializes its list of creators from the globally registered ones.
func NewVectorStoreFactory() *VectorStoreFactory {
	globalCreatorsMu.RLock()
	defer globalCreatorsMu.RUnlock()

	instanceCreators := make(map[string]func(config map[string]interface{}) (vectorstores.VectorStore, error))
	for name, creator := range globalCreators {
		instanceCreators[name] = creator
	}

	return &VectorStoreFactory{
		creators: instanceCreators,
	}
}

// Create creates a new vector store instance based on the given name and configuration.
func (f *VectorStoreFactory) Create(name string, config map[string]interface{}) (vectorstores.VectorStore, error) {
	creator, ok := f.creators[name]
	if !ok {
		// For debugging, list available creators if not found
		available := make([]string, 0, len(f.creators))
		for k := range f.creators {
			available = append(available, k)
		}
		return nil, fmt.Errorf("vector store provider 	%s	 not found. Available providers: %v", name, available)
	}
	return creator(config)
}
