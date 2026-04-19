package testutil

import (
	"github.com/lookatitude/beluga-ai/v2/internal/testutil/mockembedder"
	"github.com/lookatitude/beluga-ai/v2/internal/testutil/mockstore"
	"github.com/lookatitude/beluga-ai/v2/internal/testutil/mockworkflow"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/workflow"
)

// Compile-time interface checks to ensure mocks implement their target interfaces.
var (
	_ embedding.Embedder      = (*mockembedder.MockEmbedder)(nil)
	_ vectorstore.VectorStore = (*mockstore.MockVectorStore)(nil)
	_ workflow.WorkflowStore  = (*mockworkflow.MockWorkflowStore)(nil)
)
