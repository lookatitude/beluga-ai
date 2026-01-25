# Beluga AI Framework - Migration Guide

This guide helps you migrate between versions and from other frameworks.

## Table of Contents

1. [Version Upgrades](#version-upgrades)
2. [Migration from LangChain](#migration-from-langchain)
3. [Migration from CrewAI](#migration-from-crewai)
4. [Deprecation Notices](#deprecation-notices)

## Version Upgrades

### Pre-1.0 Versions

Beluga AI is currently in pre-1.0 development. API changes may occur.

### Breaking Changes

When upgrading, check:
- [CHANGELOG.md](https://github.com/lookatitude/beluga-ai/blob/main/CHANGELOG.md) for breaking changes
- Package READMEs for API changes
- This guide for migration steps

### Upgrade Process

1. Review CHANGELOG
2. Update dependencies: `go get -u github.com/lookatitude/beluga-ai`
3. Run tests: `go test ./...`
4. Update code based on breaking changes
5. Verify functionality

## Migration from LangChain

### Concept Mapping

| LangChain | Beluga AI |
|-----------|-----------|
| `LLM` | `llms.ChatModel` |
| `ChatModel` | `llms.ChatModel` |
| `Agent` | `agents.Agent` |
| `Memory` | `memory.Memory` |
| `VectorStore` | `vectorstores.VectorStore` |
| `Chain` | `orchestration.Chain` |

### Code Examples

#### LangChain (Python)

```python
from langchain.llms import OpenAI
llm = OpenAI()
response = llm("Hello")
```

#### Beluga AI (Go)

```go
config := llms.NewConfig(
    llms.WithProvider("openai"),
    llms.WithModelName("gpt-3.5-turbo"),
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
)
factory := llms.NewFactory()
provider, _ := factory.CreateProvider("openai", config)
response, _ := provider.Generate(ctx, messages)
```

## Migration from CrewAI

### Concept Mapping

| CrewAI | Beluga AI |
|---------|-----------|
| `Agent` | `agents.Agent` |
| `Task` | `orchestration.Task` |
| `Crew` | `orchestration.Graph` |

### Code Examples

#### CrewAI (Python)

```python
from crewai import Agent, Task, Crew

agent = Agent(role="researcher")
task = Task(description="Research topic")
crew = Crew(agents=[agent], tasks=[task])
result = crew.kickoff()
```

#### Beluga AI (Go)

```go
agent, _ := agents.NewBaseAgent("researcher", llm, tools)
task := orchestration.NewTask("Research topic", agent)
graph := orchestration.NewGraph()
graph.AddNode("task", task)
result, _ := graph.Invoke(ctx, input)
```

## Deprecation Notices

### Current Deprecations

None currently. Check [CHANGELOG.md](https://github.com/lookatitude/beluga-ai/blob/main/CHANGELOG.md) for updates.

### Deprecation Policy

- Deprecated APIs marked in documentation
- Minimum 2 versions before removal
- Migration guides provided
- Replacement APIs documented

## Getting Help

If you encounter migration issues:

1. Check [Troubleshooting Guide](./troubleshooting.md)
2. Review [CHANGELOG.md](https://github.com/lookatitude/beluga-ai/blob/main/CHANGELOG.md)
3. Search GitHub Issues
4. Create a new issue with details

---

**Last Updated:** Migration guide is maintained. Check back for updates.

