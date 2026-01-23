/**
 * Sidebar configuration for Beluga AI documentation
 * This provides a structured navigation for all documentation sections
 *
 * Structure:
 * - Documentation: Regular docs from docs/ folder (guides, concepts, tutorials)
 * - API Reference: Godoc-generated from docs/api-docs/packages/
 * - Use Cases: Real-world examples from docs/use-cases/
 */

module.exports = {
  docs: [
    'README',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'installation',
        'quickstart',
        {
          type: 'category',
          label: 'First Steps',
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
        'concepts/README',
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
        'guides/llm-providers',
        'guides/llm-streaming-tool-calls',
        'guides/agent-types',
        'guides/rag-multimodal',
        'guides/extensibility',
        'guides/implementing-providers',
        'guides/observability-tracing',
        {
          type: 'category',
          label: 'Voice',
          items: [
            'guides/voice-agents',
            'guides/voice-providers',
            'guides/voice-performance',
            'guides/voice-troubleshooting',
            'guides/s2s-implementation',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Tutorials',
      items: [
        'tutorials/README',
        {
          type: 'category',
          label: 'Foundation',
          items: [
            'tutorials/foundation/README',
            'tutorials/foundation/core-custom-runnable',
            'tutorials/foundation/core-middleware-implementation',
            'tutorials/foundation/config-environment-secrets',
            'tutorials/foundation/config-vault-integration',
            'tutorials/foundation/schema-custom-message-types',
            'tutorials/foundation/schema-modeling-multiturn-chats',
            'tutorials/foundation/monitoring-otel-tracing',
            'tutorials/foundation/monitoring-health-checks',
            'tutorials/foundation/monitoring-prometheus-grafana',
          ],
        },
        {
          type: 'category',
          label: 'Providers',
          items: [
            'tutorials/providers/README',
            'tutorials/providers/llms-new-provider',
            'tutorials/providers/llms-advanced-inference',
            'tutorials/providers/embeddings-multimodal-google',
            'tutorials/providers/embeddings-finetuning-strategies',
            'tutorials/providers/vectorstores-inmemory-local',
            'tutorials/providers/vectorstores-pgvector-sharding',
            'tutorials/providers/prompts-message-templates',
            'tutorials/providers/prompts-reusable-system-prompts',
          ],
        },
        {
          type: 'category',
          label: 'Higher-Level',
          items: [
            'tutorials/higher-level/README',
            'tutorials/higher-level/agents-research-agent',
            'tutorials/higher-level/agents-multi-agent-orchestration',
            'tutorials/higher-level/agents-tools-registry',
            'tutorials/higher-level/chatmodels-multi-provider',
            'tutorials/higher-level/chatmodels-model-switching',
            'tutorials/higher-level/memory-redis-persistence',
            'tutorials/higher-level/memory-summary-window-patterns',
            'tutorials/higher-level/retrievers-hybrid-search',
            'tutorials/higher-level/retrievers-multiquery-chains',
            'tutorials/higher-level/orchestration-dag-agents',
            'tutorials/higher-level/orchestration-message-bus',
            'tutorials/higher-level/orchestration-temporal-workflows',
            'tutorials/higher-level/server-rest-deployment',
            'tutorials/higher-level/server-mcp-tools',
            'tutorials/higher-level/safety-content-moderation',
            'tutorials/higher-level/safety-human-in-loop',
          ],
        },
        {
          type: 'category',
          label: 'Voice',
          items: [
            'tutorials/voice/README',
            'tutorials/voice/voice-stt-realtime-streaming',
            'tutorials/voice/voice-stt-whisper-finetuning',
            'tutorials/voice/voice-tts-elevenlabs-cloning',
            'tutorials/voice/voice-tts-ssml-tuning',
            'tutorials/voice/voice-vad-custom-silero',
            'tutorials/voice/voice-sensitivity-tuning',
            'tutorials/voice/voice-turn-sentence-boundary-detection',
            'tutorials/voice/voice-turn-ml-based-prediction',
            'tutorials/voice/voice-session-interruptions',
            'tutorials/voice/voice-session-preemptive-generation',
            'tutorials/voice/voice-backends-livekit-vapi',
            'tutorials/voice/voice-backend-scalable-provider',
            'tutorials/voice/voice-s2s-amazon-nova',
            'tutorials/voice/voice-s2s-reasoning-modes',
          ],
        },
        {
          type: 'category',
          label: 'Other',
          items: [
            'tutorials/other/README',
            'tutorials/other/docloaders-directory-pdf-scraper',
            'tutorials/other/docloaders-lazy-loading',
            'tutorials/other/textsplitters-markdown-chunking',
            'tutorials/other/textsplitters-semantic-splitting',
            'tutorials/other/messaging-whatsapp-bot',
            'tutorials/other/messaging-omnichannel-gateway',
            'tutorials/other/multimodal-visual-reasoning',
            'tutorials/other/multimodal-audio-analysis',
          ],
        },
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
        'cookbook/README',
        'cookbook/quick-solutions',
        {
          type: 'category',
          label: 'Agents',
          items: [
            'cookbook/agent-recipes',
            'cookbook/custom-agent',
            'cookbook/agents-parallel-step-execution',
            'cookbook/agents-handling-tool-failures-hallucinations',
          ],
        },
        {
          type: 'category',
          label: 'LLMs & Chat Models',
          items: [
            'cookbook/llm-error-handling',
            'cookbook/llms-streaming-tool-logic-handler',
            'cookbook/llms-token-counting-no-performance-hit',
            'cookbook/chatmodels-multi-step-history-trimming',
            'cookbook/chatmodels-streaming-chunks-metadata',
          ],
        },
        {
          type: 'category',
          label: 'RAG & Retrieval',
          items: [
            'cookbook/rag-recipes',
            'cookbook/document-ingestion-recipes',
            'cookbook/documentloaders-parallel-file-walkers',
            'cookbook/documentloaders-robust-error-handling-corrupt-docs',
            'cookbook/retrievers-parent-document-retrieval',
            'cookbook/retrievers-reranking-cohere-rerank',
          ],
        },
        {
          type: 'category',
          label: 'Embeddings & Vector Stores',
          items: [
            'cookbook/embeddings-batch-optimization',
            'cookbook/embeddings-metadata-aware-clusters',
            'cookbook/vectorstores-advanced-meta-filtering',
            'cookbook/vectorstores-reindexing-status-tracking',
          ],
        },
        {
          type: 'category',
          label: 'Memory',
          items: [
            'cookbook/memory-recipes',
            'cookbook/memory-ttl-cleanup-strategies',
            'cookbook/memory-window-based-context-recovery',
          ],
        },
        {
          type: 'category',
          label: 'Tools & Integration',
          items: [
            'cookbook/tool-recipes',
            'cookbook/integration-recipes',
          ],
        },
        {
          type: 'category',
          label: 'Core & Config',
          items: [
            'cookbook/core-global-retry-wrappers',
            'cookbook/core-advanced-context-timeout-management',
            'cookbook/config-hot-reloading-production',
            'cookbook/config-masking-secrets-logs',
          ],
        },
        {
          type: 'category',
          label: 'Prompts & Schema',
          items: [
            'cookbook/prompts-dynamic-message-chain-templates',
            'cookbook/prompts-partial-variable-substitution',
            'cookbook/schema-custom-validation-middleware',
            'cookbook/schema-recursive-schema-handling',
          ],
        },
        {
          type: 'category',
          label: 'Text Splitters',
          items: [
            'cookbook/textsplitters-advanced-code-splitting-tree-sitter',
            'cookbook/textsplitters-sentence-boundary-aware',
          ],
        },
        {
          type: 'category',
          label: 'Orchestration',
          items: [
            'cookbook/orchestration-parallel-node-execution',
            'cookbook/orchestration-workflow-checkpointing',
          ],
        },
        {
          type: 'category',
          label: 'Monitoring & Safety',
          items: [
            'cookbook/monitoring-custom-metrics-s2s-voice',
            'cookbook/monitoring-trace-aggregation-multi-agents',
            'cookbook/safety-mitigating-prompt-injection-regex',
            'cookbook/safety-pii-redaction-logs',
          ],
        },
        {
          type: 'category',
          label: 'Messaging & Multimodal',
          items: [
            'cookbook/messaging-conversation-expiry-logic',
            'cookbook/messaging-handling-inbound-media',
            'cookbook/multimodal-capability-based-fallbacks',
            'cookbook/multimodal-processing-multiple-images-per-prompt',
          ],
        },
        {
          type: 'category',
          label: 'Server',
          items: [
            'cookbook/server-correlating-request-ids',
            'cookbook/server-rate-limiting-per-project',
          ],
        },
        {
          type: 'category',
          label: 'Voice',
          items: [
            'cookbook/voice-backends',
            'cookbook/voice-backend-scaling-concurrent-streams',
            'cookbook/voice-session-preemptive-generation',
            'cookbook/voice-session-long-utterances',
            'cookbook/voice-stt-jitter-buffer-management',
            'cookbook/voice-stt-overcoming-background-noise',
            'cookbook/voice-tts-ssml-emphasis-pause-tuning',
            'cookbook/voice-tts-multi-speaker-dialogue-synthesis',
            'cookbook/voice-vad-sensitivity-profiles',
            'cookbook/voice-turn-sentence-boundary-aware',
            'cookbook/voice-turn-ml-based-barge-in',
            'cookbook/voice-s2s-minimizing-glass-to-glass-latency',
            'cookbook/voice-s2s-handling-speech-interruption',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Integrations',
      items: [
        'integrations/README',
        {
          type: 'category',
          label: 'Agents',
          items: [
            'integrations/agents/agents-mcp-tools-integration',
            'integrations/agents/agents-custom-tools-registry',
          ],
        },
        {
          type: 'category',
          label: 'Chat Models',
          items: [
            'integrations/chatmodels/openai-assistants-api-bridge',
            'integrations/chatmodels/custom-mock-ui-testing',
          ],
        },
        {
          type: 'category',
          label: 'LLMs',
          items: [
            'integrations/llms/anthropic-claude-enterprise',
            'integrations/llms/aws-bedrock-integration',
          ],
        },
        {
          type: 'category',
          label: 'Memory',
          items: [
            'integrations/memory/redis-distributed-locking',
            'integrations/memory/mongodb-context-persistence',
          ],
        },
        {
          type: 'category',
          label: 'Config',
          items: [
            'integrations/config/hashicorp-vault-connector',
            'integrations/config/viper-environment-overrides',
          ],
        },
        {
          type: 'category',
          label: 'Core',
          items: [
            'integrations/core/context-deep-dive',
            'integrations/core/zap-logrus-providers',
          ],
        },
        {
          type: 'category',
          label: 'Document Loaders',
          items: [
            'integrations/docloaders/google-drive-api-scraper',
            'integrations/docloaders/aws-s3-event-driven-loader',
          ],
        },
        {
          type: 'category',
          label: 'Embeddings',
          items: [
            'integrations/embeddings/cohere-multilingual-embedder',
            'integrations/embeddings/ollama-local-embeddings',
          ],
        },
        {
          type: 'category',
          label: 'Messaging',
          items: [
            'integrations/messaging/twilio-conversations-api',
            'integrations/messaging/slack-webhook-handler',
          ],
        },
        {
          type: 'category',
          label: 'Monitoring',
          items: [
            'integrations/monitoring/datadog-dashboard-templates',
            'integrations/monitoring/langsmith-debugging-integration',
          ],
        },
        {
          type: 'category',
          label: 'Multimodal',
          items: [
            'integrations/multimodal/google-vertex-ai-vision',
            'integrations/multimodal/pixtral-mistral-integration',
          ],
        },
        {
          type: 'category',
          label: 'Orchestration',
          items: [
            'integrations/orchestration/kubernetes-job-scheduler',
            'integrations/orchestration/nats-message-bus',
          ],
        },
        {
          type: 'category',
          label: 'Prompts',
          items: [
            'integrations/prompts/langchain-hub-loading',
            'integrations/prompts/local-filesystem-template-store',
          ],
        },
        {
          type: 'category',
          label: 'Retrievers',
          items: [
            'integrations/retrievers/elasticsearch-keyword-search',
            'integrations/retrievers/weaviate-rag-connector',
          ],
        },
        {
          type: 'category',
          label: 'Safety',
          items: [
            'integrations/safety/safety-result-json-reporting',
            'integrations/safety/third-party-ethical-api-filter',
          ],
        },
        {
          type: 'category',
          label: 'Schema',
          items: [
            'integrations/schema/json-schema-validation',
            'integrations/schema/pydantic-go-struct-bridge',
          ],
        },
        {
          type: 'category',
          label: 'Server',
          items: [
            'integrations/server/kubernetes-helm-deployment',
            'integrations/server/auth0-jwt-authentication',
          ],
        },
        {
          type: 'category',
          label: 'Text Splitters',
          items: [
            'integrations/textsplitters/spacy-sentence-tokenizer',
            'integrations/textsplitters/tiktoken-byte-pair-encoding',
          ],
        },
        {
          type: 'category',
          label: 'Vector Stores',
          items: [
            'integrations/vectorstores/pinecone-serverless',
            'integrations/vectorstores/qdrant-cloud-cluster',
          ],
        },
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
              label: 'STT',
              items: [
                'integrations/voice/stt/deepgram-live-streams',
                'integrations/voice/stt/amazon-transcribe-websockets',
              ],
            },
            {
              type: 'category',
              label: 'TTS',
              items: [
                'integrations/voice/tts/elevenlabs-streaming-api',
                'integrations/voice/tts/azure-cognitive-services-speech',
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
            {
              type: 'category',
              label: 'S2S',
              items: [
                'integrations/voice/s2s/openai-realtime-api',
                'integrations/voice/s2s/amazon-nova-bedrock-streaming',
              ],
            },
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/architecture',
        'architecture/component-diagrams',
        'architecture/data-flows',
        'architecture/sequences',
        'architecture/gaps-analysis',
        'architecture/import-cycle-audit',
        'architecture/test-import-cycles',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api-reference',
        {
          type: 'category',
          label: 'Core Packages',
          items: [
            'api-docs/packages/agents',
            'api-docs/packages/chatmodels',
            'api-docs/packages/config',
            'api-docs/packages/core',
            'api-docs/packages/embeddings',
            'api-docs/packages/llms',
            'api-docs/packages/memory',
            'api-docs/packages/monitoring',
            'api-docs/packages/orchestration',
            'api-docs/packages/prompts',
            'api-docs/packages/retrievers',
            'api-docs/packages/schema',
            'api-docs/packages/server',
            'api-docs/packages/vectorstores',
            'api-docs/packages/tools',
          ],
        },
        {
          type: 'category',
          label: 'LLM Providers',
          items: [
            'api-docs/packages/llms/anthropic',
            'api-docs/packages/llms/bedrock',
            'api-docs/packages/llms/mock',
            'api-docs/packages/llms/ollama',
            'api-docs/packages/llms/openai',
          ],
        },
        {
          type: 'category',
          label: 'Voice Packages',
          items: [
            'api-docs/packages/voice/stt',
            'api-docs/packages/voice/tts',
            'api-docs/packages/voice/vad',
            'api-docs/packages/voice/turndetection',
            'api-docs/packages/voice/transport',
            'api-docs/packages/voice/noise',
            'api-docs/packages/voice/session',
          ],
        },
        'api-docs/packages/rag',
      ],
    },
    {
      type: 'category',
      label: 'Tools',
      items: [
        'tools/test-analyzer',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      items: [
        'best-practices',
        'package_design_patterns',
        'framework-comparison',
        'api-package-inventory',
        'troubleshooting',
      ],
    },
    {
      type: 'category',
      label: 'Use Cases',
      items: [
        'use-cases/README',
      ],
    },
  ],
};
