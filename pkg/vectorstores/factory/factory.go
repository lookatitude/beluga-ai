package factory

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

// VectorStoreFactory is responsible for creating vector store instances.
// It allows for easy addition of new vector store types.
type VectorStoreFactory struct {
	creators map[string]func(config map[string]interface{}) (vectorstores.VectorStore, error)
}

// NewVectorStoreFactory creates a new instance of VectorStoreFactory.
func NewVectorStoreFactory() *VectorStoreFactory {
	return &VectorStoreFactory{
		creators: make(map[string]func(config map[string]interface{}) (vectorstores.VectorStore, error)),
	}
}

// Register adds a new vector store creator to the factory.
func (f *VectorStoreFactory) Register(name string, creator func(config map[string]interface{}) (vectorstores.VectorStore, error)) {
	f.creators[name] = creator
}

// Create creates a new vector store instance based on the given name and configuration.
func (f *VectorStoreFactory) Create(name string, config map[string]interface{}) (vectorstores.VectorStore, error) {
	creator, ok := f.creators[name]
	if !ok {
		return nil, fmt.Errorf("vector store %s not found", name)
	}
	return creator(config)
}

