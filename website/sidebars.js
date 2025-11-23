/**
 * Sidebar configuration for Beluga AI documentation
 * This provides a structured navigation for all documentation sections
 */

module.exports = {
  docs: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/quickstart',
        {
          type: 'category',
          label: 'Tutorials',
          items: [
            'getting-started/tutorials/first-llm-call',
            'getting-started/tutorials/simple-rag',
            'getting-started/tutorials/first-agent',
            'getting-started/tutorials/working-with-tools',
            'getting-started/tutorials/memory-management',
            'getting-started/tutorials/orchestration-basics',
            'getting-started/tutorials/production-deployment',
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
      ],
    },
    {
      type: 'category',
      label: 'Voice Agents',
      items: [
        'voice/index',
        'voice/stt',
        'voice/tts',
        'voice/vad',
        'voice/turndetection',
        'voice/transport',
        'voice/noise',
        'voice/session',
      ],
    },
    {
      type: 'category',
      label: 'Guides',
      items: [
        'guides/architecture',
        'guides/best-practices',
        'guides/troubleshooting',
        'guides/migration',
        'guides/package-design-patterns',
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
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      items: [
        'use-cases/index',
        'use-cases/enterprise-rag-knowledge-base',
        'use-cases/multi-agent-customer-support',
        'use-cases/intelligent-document-processing',
        'use-cases/real-time-data-analysis-agent',
        'use-cases/conversational-ai-assistant',
        'use-cases/automated-code-review-system',
        'use-cases/distributed-workflow-orchestration',
        'use-cases/semantic-search-recommendation',
        'use-cases/multi-model-llm-gateway',
        'use-cases/production-agent-platform',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/index',
        {
          type: 'category',
          label: 'Packages',
          items: [
            'api/packages/core',
            'api/packages/config',
            'api/packages/schema',
            'api/packages/llms_base',
            {
              type: 'category',
              label: 'LLM Providers',
              items: [
                'api/packages/llms/anthropic',
                'api/packages/llms/bedrock',
                'api/packages/llms/cohere',
                'api/packages/llms/ollama',
                'api/packages/llms/openai',
              ],
            },
            'api/packages/agents',
            'api/packages/chatmodels',
            'api/packages/tools',
            'api/packages/memory',
            'api/packages/embeddings',
            'api/packages/vectorstores',
            'api/packages/retrievers',
            'api/packages/rag',
            'api/packages/prompts',
            'api/packages/orchestration',
            'api/packages/monitoring',
            'api/packages/server',
            {
              type: 'category',
              label: 'Voice Packages',
              items: [
                'api/packages/voice/stt',
                'api/packages/voice/tts',
                'api/packages/voice/vad',
                'api/packages/voice/turndetection',
                'api/packages/voice/transport',
                'api/packages/voice/noise',
                'api/packages/voice/session',
              ],
            },
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'reference/framework-comparison',
        'reference/documentation-roadmap',
      ],
    },
  ],
};
