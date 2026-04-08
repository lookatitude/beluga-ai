// Package procedural implements the 4th memory tier for Beluga AI agents:
// procedural memory (how-to knowledge).
//
// Procedural memory stores [schema.Skill] entries — structured, goal-directed
// processes that agents learn from successful task completions. Unlike archival
// memory which stores raw content, procedural memory stores abstracted
// procedures that can be retrieved and reused across contexts.
//
// Skills are NOT executable code. They are structured descriptions of steps,
// triggers, and metadata that an agent can reference when encountering similar
// tasks. This design avoids the security risks of storing and executing
// arbitrary code.
//
// # Architecture
//
// The package provides three key components:
//
//   - [ProceduralMemory]: the main CRUD + search interface backed by vector
//     embeddings for semantic skill retrieval.
//   - [SkillExtractor]: an interface for extracting skills from execution traces,
//     with a built-in [LLMExtractor] that uses a ChatModel.
//   - [InMemoryStore]: a thread-safe in-memory skill store for testing.
//
// # Usage
//
//	pm := procedural.New(embedder, vectorStore)
//	err := pm.SaveSkill(ctx, &schema.Skill{
//	    Name:        "deploy-service",
//	    Description: "Deploy a microservice to Kubernetes",
//	    Steps:       []string{"build image", "push to registry", "apply manifests"},
//	    Triggers:    []string{"deploy", "release", "ship"},
//	    Confidence:  0.8,
//	})
//
//	skills, err := pm.SearchSkills(ctx, "how to deploy", 5)
//
// # Hooks
//
// All operations support optional hooks for observability:
//
//	pm := procedural.New(embedder, vectorStore,
//	    procedural.WithHooks(procedural.Hooks{
//	        OnSkillSaved: func(ctx context.Context, skill *schema.Skill) {
//	            log.Printf("saved skill: %s", skill.Name)
//	        },
//	    }),
//	)
//
// # Skill Extraction
//
// The [LLMExtractor] uses a ChatModel to abstract procedures from execution
// traces. It should be wired as a post-save hook after successful completions:
//
//	extractor := procedural.NewLLMExtractor(chatModel)
//	skill, err := extractor.Extract(ctx, input, output, metadata)
package procedural
