/**
 * Sidebar configuration for Beluga AI documentation
 * This provides a structured navigation for all documentation sections
 */

module.exports = {
  docs: [
    'README',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'INSTALLATION',
        'QUICKSTART',
        {
          type: 'category',
          label: 'Tutorials',
          items: [
            'getting-started/01-first-llm-call',
            'getting-started/02-simple-rag',
            'getting-started/03-first-agent',
            'getting-started/04-working-with-tools',
            'getting-started/05-memory-management',
            'getting-started/06-orchestration-basics',
            'getting-started/07-production-deployment',
            'getting-started/03-document-ingestion',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Concepts',
      items: [
        'concepts/core',
        'concepts/llms',
        'concepts/agents',
        'concepts/memory',
        'concepts/orchestration',
        'concepts/rag',
        'concepts/document-loading',
        'concepts/text-splitting',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        // Core Architecture & Patterns
        'architecture',
        'BEST_PRACTICES',
        'package_design_patterns',
        // Advanced Features (User Story 1)
        {
          type: 'category',
          label: 'Advanced Features',
          items: [
            'guides/llm-streaming-tool-calls',
            'guides/agent-types',
            'guides/rag-multimodal',
          ],
        },
        // Provider Integration (User Story 2)
        {
          type: 'category',
          label: 'Provider Integration',
          items: [
            'guides/llm-providers',
            'guides/voice-providers',
            'guides/extensibility',
            'guides/implementing-providers',
          ],
        },
        // Production Deployment (User Story 3)
        {
          type: 'category',
          label: 'Production Deployment',
          items: [
            'guides/observability-tracing',
            'guides/voice-performance',
            'guides/voice-troubleshooting',
          ],
        },
        // Voice & S2S
        {
          type: 'category',
          label: 'Voice',
          items: [
            'guides/voice-agents',
            'guides/s2s-implementation',
          ],
        },
        // Migration & Troubleshooting
        'MIGRATION',
        'TROUBLESHOOTING',
      ],
    },
    {
      type: 'category',
      label: 'Providers',
      items: [
        {
          type: 'category',
          label: 'LLMs',
          items: [
            'providers/llms/openai',
            'providers/llms/anthropic',
            'providers/llms/ollama',
            'providers/llms/comparison',
          ],
        },
        {
          type: 'category',
          label: 'Embeddings',
          items: [
            'providers/embeddings/openai',
            'providers/embeddings/ollama',
            'providers/embeddings/selection',
          ],
        },
        {
          type: 'category',
          label: 'Vector Stores',
          items: [
            'providers/vectorstores/pgvector',
            'providers/vectorstores/comparison',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Cookbook',
      items: [
        'cookbook/quick-solutions',
        'cookbook/agent-recipes',
        'cookbook/rag-recipes',
        'cookbook/memory-recipes',
        'cookbook/tool-recipes',
        'cookbook/integration-recipes',
        'cookbook/document-ingestion-recipes',
        // Advanced Recipes (added in docs gap analysis)
        {
          type: 'category',
          label: 'Advanced Recipes',
          items: [
            'cookbook/llm-error-handling',
            'cookbook/custom-agent',
            'cookbook/voice-backends',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      items: [
        'use-cases/README',
        // Core Use Cases
        'use-cases/01-enterprise-rag-knowledge-base',
        'use-cases/02-multi-agent-customer-support',
        'use-cases/03-intelligent-document-processing',
        'use-cases/04-real-time-data-analysis-agent',
        'use-cases/05-conversational-ai-assistant',
        'use-cases/06-automated-code-review-system',
        'use-cases/07-distributed-workflow-orchestration',
        'use-cases/08-semantic-search-recommendation',
        'use-cases/09-multi-model-llm-gateway',
        'use-cases/10-production-agent-platform',
        // Advanced Use Cases (added in docs gap analysis)
        {
          type: 'category',
          label: 'Advanced Scenarios',
          items: [
            'use-cases/11-batch-processing',
            'use-cases/monitoring-dashboards',
            'use-cases/voice-sessions',
            'use-cases/rag-strategies',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'FRAMEWORK_COMPARISON',
        'DOCUMENTATION_ROADMAP',
        'API_PACKAGE_INVENTORY',
        'GODOC_COVERAGE_REPORT',
      ],
    },
  ],
};
