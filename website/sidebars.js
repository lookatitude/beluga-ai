/**
 * Sidebar configuration for Beluga AI documentation
 * This provides a structured navigation for all documentation sections
 * 
 * Structure:
 * - Documentation: Regular docs from docs/ folder (guides, concepts, tutorials)
 * - API Reference: Godoc-generated from website/docs/api/packages/
 * - Use Cases: Real-world examples from docs/use-cases/
 */

module.exports = {
  docs: [
    'README',
    {
      type: 'category',
      label: 'Documentation',
      items: [
        {
          type: 'category',
          label: 'Getting Started',
          items: [
            'installation',
            'quickstart',
            {
              type: 'category',
              label: 'Tutorials',
              items: [
                'getting-started/first-llm-call',
                'getting-started/simple-rag',
                'getting-started/first-agent',
                'getting-started/document-ingestion',
                'getting-started/working-with-tools',
                'getting-started/memory-management',
                'getting-started/orchestration-basics',
                'getting-started/production-deployment',
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
            'best-practices',
            'package_design_patterns',
            // Advanced Features
            {
              type: 'category',
              label: 'Advanced Features',
              items: [
                'guides/llm-streaming-tool-calls',
                'guides/agent-types',
                'guides/rag-multimodal',
              ],
            },
            // Provider Integration
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
            // Production Deployment
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
            'migration',
            'troubleshooting',
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
            // Advanced Recipes
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
          label: 'Reference',
          items: [
            'framework-comparison',
            'documentation-roadmap',
            'api-package-inventory',
            'godoc-coverage-report',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/index',
        {
          type: 'category',
          label: 'Core Packages',
          items: [
            'api/packages/core',
            'api/packages/schema',
            'api/packages/config',
          ],
        },
        {
          type: 'category',
          label: 'LLM Packages',
          items: [
            'api/packages/llms_base',
            'api/packages/llms',
            'api/packages/chatmodels',
            {
              type: 'category',
              label: 'LLM Providers',
              items: [
                'api/packages/llms/openai',
                'api/packages/llms/anthropic',
                'api/packages/llms/bedrock',
                'api/packages/llms/ollama',
                'api/packages/llms/mock',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'Agent Packages',
          items: [
            'api/packages/agents',
            'api/packages/tools',
            'api/packages/orchestration',
          ],
        },
        {
          type: 'category',
          label: 'Memory & RAG Packages',
          items: [
            'api/packages/memory',
            'api/packages/rag',
            'api/packages/embeddings',
            'api/packages/vectorstores',
            'api/packages/retrievers',
          ],
        },
        {
          type: 'category',
          label: 'Voice Packages',
          items: [
            {
              type: 'category',
              label: 'Voice Components',
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
        {
          type: 'category',
          label: 'Supporting Packages',
          items: [
            'api/packages/prompts',
            'api/packages/monitoring',
            'api/packages/server',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      items: [
        'use-cases/README',
        // Note: Document IDs may not match filenames with numeric prefixes
        // These will be verified during build
      ],
    },
  ],
};
