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
            'concepts/agent-design',
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
            'architecture/architecture',
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
            {
              type: 'category',
              label: 'Advanced Recipes',
              items: [
                'cookbook/llm-error-handling',
                'cookbook/custom-agent',
                'cookbook/voice-backends',
                'cookbook/voice-backend-scaling-concurrent-streams',
                'cookbook/voice-session-preemptive-generation',
                'cookbook/voice-session-long-utterances',
                'cookbook/voice-turn-sentence-boundary-aware',
                'cookbook/voice-turn-ml-based-barge-in',
                'cookbook/voice-vad-sensitivity-profiles',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'Integrations',
          items: [
            {
              type: 'category',
              label: 'Voice',
              items: [
                {
                  type: 'category',
                  label: 'Backend',
                  items: [
                    'integrations/voice/backend/livekit-webhooks-integration',
                    'integrations/voice/backend/vapi-custom-tools',
                  ],
                },
                {
                  type: 'category',
                  label: 'Session',
                  items: [
                    'integrations/voice/session/voice-session-persistence',
                    'integrations/voice/session/multi-provider-session-routing',
                  ],
                },
                {
                  type: 'category',
                  label: 'VAD',
                  items: [
                    'integrations/voice/vad/webrtc-vad-browser',
                    'integrations/voice/vad/onnx-runtime-edge-vad',
                  ],
                },
                {
                  type: 'category',
                  label: 'Turn Detection',
                  items: [
                    'integrations/voice/turn/custom-turn-detectors-noisy-environments',
                    'integrations/voice/turn/heuristic-tuning',
                  ],
                },
              ],
            },
            {
              type: 'category',
              label: 'Agents',
              items: [
                'integrations/agents/agents-mcp-tools-integration',
                'integrations/agents/agents-custom-tools-registry',
              ],
            },
          ],
        },
        {
          type: 'category',
          label: 'Reference',
          items: [
            'framework-comparison',
            'api-package-inventory',
          ],
        },
      ],
    },
    {
      type: 'doc',
      id: 'api-reference',
      label: 'API Reference',
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
