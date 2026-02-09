package testutil

import (
	"github.com/lookatitude/beluga-ai/internal/testutil/mockembedder"
	"github.com/lookatitude/beluga-ai/internal/testutil/mockstore"
	"github.com/lookatitude/beluga-ai/internal/testutil/mockworkflow"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/workflow"
)

// Compile-time interface checks to ensure mocks implement their target interfaces.
var (
	_ embedding.Embedder      = (*mockembedder.MockEmbedder)(nil)
	_ vectorstore.VectorStore = (*mockstore.MockVectorStore)(nil)
	_ workflow.WorkflowStore  = (*mockworkflow.MockWorkflowStore)(nil)
)
