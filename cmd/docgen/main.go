// Command docgen generates Starlight-compatible Markdown API reference pages
// from Go package doc.go files in the Beluga AI v2 codebase.
//
// Usage:
//
//	go run ./cmd/docgen
//
// It reads all doc.go files, parses the godoc comments, groups packages by
// category, and writes organized Markdown files to
// docs/website/src/content/docs/api-reference/.
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// page represents a single generated Markdown page.
type page struct {
	Filename    string // output filename (e.g. "core.md")
	Title       string // page title
	Description string // frontmatter description
	Packages    []packageGroup
}

// packageGroup is a package or set of packages included on one page.
type packageGroup struct {
	ImportPath string // e.g. "github.com/lookatitude/beluga-ai/core"
	PkgName    string // e.g. "core"
	DocComment string // raw godoc comment text from doc.go
	IsProvider bool   // whether this is a provider sub-package
}

const (
	modulePath = "github.com/lookatitude/beluga-ai"
	outputDir  = "docs/website/src/content/docs/api-reference"
)

func main() {
	root, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error finding module root: %v\n", err)
		os.Exit(1)
	}

	// Collect all doc.go files
	pkgs, err := collectDocFiles(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error collecting doc.go files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d packages with doc.go files\n", len(pkgs))

	// Build pages from the package map
	pages := buildPages(pkgs)

	// Write output
	outDir := filepath.Join(root, outputDir)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	for _, p := range pages {
		content := renderPage(p)
		path := filepath.Join(outDir, p.Filename)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("  wrote %s (%d packages)\n", p.Filename, len(p.Packages))
	}

	// Write index page
	indexContent := renderIndex(pages)
	indexPath := filepath.Join(outDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing index: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  wrote index.md\n")

	fmt.Printf("\nGenerated %d API reference pages + index\n", len(pages))
}

// findModuleRoot walks up from cwd to find go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod")
		}
		dir = parent
	}
}

// collectDocFiles finds all doc.go files and extracts package doc comments.
func collectDocFiles(root string) (map[string]packageGroup, error) {
	pkgs := make(map[string]packageGroup)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		// Skip vendor, .git, node_modules, docs
		base := filepath.Base(path)
		if info.IsDir() && (base == "vendor" || base == ".git" || base == "node_modules" || base == "docs" || base == "examples") {
			return filepath.SkipDir
		}
		if base != "doc.go" {
			return nil
		}

		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return nil
		}

		// Skip internal packages
		if strings.Contains(rel, "internal/") || rel == "internal" {
			return nil
		}

		importPath := modulePath + "/" + filepath.ToSlash(rel)
		if rel == "." {
			importPath = modulePath
		}

		doc, pkgName, err := extractDocComment(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
			return nil
		}

		isProvider := strings.Contains(rel, "/providers/") || strings.Contains(rel, "/stores/") || strings.Contains(rel, "/adapters/")

		pkgs[rel] = packageGroup{
			ImportPath: importPath,
			PkgName:    pkgName,
			DocComment: doc,
			IsProvider: isProvider,
		}
		return nil
	})

	return pkgs, err
}

// extractDocComment parses a doc.go file and returns the package doc comment and package name.
func extractDocComment(path string) (string, string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return "", "", err
	}

	pkgName := ""
	if f.Name != nil {
		pkgName = f.Name.Name
	}

	// Get the package doc comment
	var doc string
	if f.Doc != nil {
		doc = f.Doc.Text()
	} else {
		// Try to get from comments associated with the package clause
		for _, cg := range f.Comments {
			if cg.Pos() < f.Package {
				doc = cg.Text()
			}
		}
	}

	return doc, pkgName, nil
}

// godocToMarkdown converts a godoc comment string to Markdown.
func godocToMarkdown(doc string) string {
	if doc == "" {
		return ""
	}

	lines := strings.Split(doc, "\n")
	var out []string
	inCode := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Godoc section headers: "# SectionName" becomes "## SectionName"
		if strings.HasPrefix(line, "# ") {
			if inCode {
				out = append(out, "```")
				inCode = false
			}
			out = append(out, "##"+line[1:])
			continue
		}

		// Code block detection: lines starting with tab or spaces that look like code
		if isCodeLine(line) {
			if !inCode {
				out = append(out, "```go")
				inCode = true
			}
			// Remove the leading tab
			trimmed := strings.TrimPrefix(line, "\t")
			out = append(out, trimmed)
			continue
		}

		// End code block — but peek ahead for continuation
		if inCode && !isCodeLine(line) {
			if line == "" && i+1 < len(lines) && isCodeLine(lines[i+1]) {
				// Blank line between two code blocks — keep it open
				out = append(out, "")
				continue
			}
			out = append(out, "```")
			inCode = false
		}

		// List items: "  - item" -> "- item"
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "   - ") {
			out = append(out, strings.TrimLeft(line, " "))
			continue
		}

		// List continuation: "    continued text" -> "  continued text"
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "    -") {
			out = append(out, "  "+strings.TrimLeft(line, " "))
			continue
		}

		// Godoc links: [TypeName] -> `TypeName`
		processed := processGodocLinks(line)
		out = append(out, processed)
	}

	if inCode {
		out = append(out, "```")
	}

	// Clean up multiple blank lines
	result := strings.Join(out, "\n")
	result = multiBlankRe.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}

// isCodeLine checks if a line is a code block line in godoc format.
// In doc.Text() output, code blocks use tab indentation while list
// continuations use space indentation. Only tab-indented lines are code.
func isCodeLine(line string) bool {
	if line == "" {
		return false
	}
	return strings.HasPrefix(line, "\t")
}

var multiBlankRe = regexp.MustCompile(`\n{3,}`)

// godocLinkRe matches [TypeName] or [Type.Method] references in godoc text.
// Go regexp doesn't support lookahead, so we match broadly and filter in the
// replacer function to skip Markdown links like [text](url).
var godocLinkRe = regexp.MustCompile(`\[([A-Z]\w*(?:\.\w+)?)\]`)

// processGodocLinks converts [TypeName] references to inline code, skipping
// Markdown-style links [text](url).
func processGodocLinks(line string) string {
	return godocLinkRe.ReplaceAllStringFunc(line, func(match string) string {
		// Find position of this match in the line
		idx := strings.Index(line, match)
		after := idx + len(match)
		if after < len(line) && line[after] == '(' {
			// This is a Markdown link [text](url), don't convert
			return match
		}
		// Extract the name and wrap in backticks
		name := match[1 : len(match)-1]
		return "`" + name + "`"
	})
}

// buildPages groups packages into page definitions.
func buildPages(pkgs map[string]packageGroup) []page {
	var pages []page

	// Define page groupings: output filename -> list of package relative paths
	type pageSpec struct {
		filename    string
		title       string
		description string
		paths       []string // relative paths to include (in order)
	}

	// Collect all known paths
	allPaths := make([]string, 0, len(pkgs))
	for p := range pkgs {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	// Helper to collect paths matching a prefix
	pathsWithPrefix := func(prefix string) []string {
		var result []string
		for _, p := range allPaths {
			if strings.HasPrefix(p, prefix) {
				result = append(result, p)
			}
		}
		return result
	}

	// Helper to get provider paths under a prefix
	providerPaths := func(prefix string) []string {
		var result []string
		for _, p := range allPaths {
			if strings.HasPrefix(p, prefix) && (strings.Contains(p, "/providers/") || strings.Contains(p, "/stores/") || strings.Contains(p, "/adapters/")) {
				result = append(result, p)
			}
		}
		return result
	}

	// Helper to get only the main package (not providers)
	mainPath := func(path string) []string {
		if _, ok := pkgs[path]; ok {
			return []string{path}
		}
		return nil
	}

	specs := []pageSpec{
		// Foundation
		{"core.md", "Core Package", "Foundation primitives: streams, Runnable, events, errors, lifecycle, multi-tenancy", mainPath("core")},
		{"schema.md", "Schema Package", "Shared types: messages, content parts, tool definitions, documents, events, sessions", mainPath("schema")},
		{"config.md", "Config Package", "Configuration loading, validation, environment variables, and hot-reload", mainPath("config")},

		// LLM
		{"llm.md", "LLM Package", "ChatModel interface, provider registry, middleware, hooks, structured output, routing", mainPath("llm")},
		{"llm-providers.md", "LLM Providers", "All LLM provider implementations: OpenAI, Anthropic, Google, Ollama, Bedrock, and more", providerPaths("llm/")},

		// Agent
		{"agent.md", "Agent Package", "Agent runtime, BaseAgent, Executor, Planner strategies, handoffs, and event bus", mainPath("agent")},
		{"agent-workflow.md", "Agent Workflows", "Sequential, Parallel, and Loop workflow agents for multi-agent orchestration", mainPath("agent/workflow")},

		// Tool
		{"tool.md", "Tool Package", "Tool interface, FuncTool, registry, MCP client integration, and middleware", mainPath("tool")},

		// Memory
		{"memory.md", "Memory Package", "MemGPT-inspired 3-tier memory: Core, Recall, Archival, graph memory, composite", mainPath("memory")},
		{"memory-stores.md", "Memory Store Providers", "Memory store implementations: in-memory, Redis, PostgreSQL, SQLite, MongoDB, Neo4j, Memgraph, Dragonfly", providerPaths("memory/")},

		// RAG
		{"rag-embedding.md", "RAG Embedding", "Embedder interface for converting text to vector embeddings", mainPath("rag/embedding")},
		{"rag-embedding-providers.md", "Embedding Providers", "Embedding provider implementations: OpenAI, Cohere, Google, Jina, Mistral, Ollama, Voyage, and more", providerPaths("rag/embedding/")},
		{"rag-vectorstore.md", "RAG Vector Store", "VectorStore interface for similarity search over document embeddings", mainPath("rag/vectorstore")},
		{"rag-vectorstore-providers.md", "Vector Store Providers", "Vector store implementations: pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, and more", providerPaths("rag/vectorstore/")},
		{"rag-retriever.md", "RAG Retriever", "Retriever strategies: Vector, Hybrid, HyDE, CRAG, Multi-Query, Ensemble, Rerank, Adaptive", mainPath("rag/retriever")},
		{"rag-loader.md", "RAG Document Loaders", "Document loaders for files, cloud storage, APIs, and web content", func() []string {
			result := mainPath("rag/loader")
			result = append(result, providerPaths("rag/loader/")...)
			return result
		}()},
		{"rag-splitter.md", "RAG Text Splitters", "Text splitting strategies for chunking documents", mainPath("rag/splitter")},

		// Voice
		{"voice.md", "Voice Package", "Frame-based voice pipeline, VAD, hybrid cascade/S2S switching", mainPath("voice")},
		{"voice-stt.md", "Voice STT", "Speech-to-text interface and providers: Deepgram, AssemblyAI, Whisper, Groq, ElevenLabs, Gladia", func() []string {
			result := mainPath("voice/stt")
			result = append(result, providerPaths("voice/stt/")...)
			return result
		}()},
		{"voice-tts.md", "Voice TTS", "Text-to-speech interface and providers: ElevenLabs, Cartesia, PlayHT, Fish, Groq, LMNT, Smallest", func() []string {
			result := mainPath("voice/tts")
			result = append(result, providerPaths("voice/tts/")...)
			return result
		}()},
		{"voice-s2s.md", "Voice S2S", "Speech-to-speech interface and providers: OpenAI Realtime, Gemini Live, Nova S2S", func() []string {
			result := mainPath("voice/s2s")
			result = append(result, providerPaths("voice/s2s/")...)
			return result
		}()},
		{"voice-transport.md", "Voice Transport", "Transport layer for voice sessions: WebSocket, LiveKit, Daily, Pipecat", func() []string {
			result := mainPath("voice/transport")
			result = append(result, providerPaths("voice/transport/")...)
			return result
		}()},
		{"voice-vad.md", "Voice VAD", "Voice activity detection providers: Silero, WebRTC", func() []string {
			var result []string
			for _, p := range allPaths {
				if strings.HasPrefix(p, "voice/vad/") {
					result = append(result, p)
				}
			}
			return result
		}()},

		// Infrastructure
		{"guard.md", "Guard Package", "Three-stage safety pipeline: input, output, tool guards with built-in and external providers", func() []string {
			result := mainPath("guard")
			result = append(result, providerPaths("guard/")...)
			return result
		}()},
		{"resilience.md", "Resilience Package", "Circuit breaker, hedge, retry, and rate limiting patterns", mainPath("resilience")},
		{"cache.md", "Cache Package", "Exact, semantic, and prompt caching with pluggable backends", func() []string {
			result := mainPath("cache")
			result = append(result, providerPaths("cache/")...)
			return result
		}()},
		{"hitl.md", "HITL Package", "Human-in-the-loop: confidence-based approval, escalation policies", mainPath("hitl")},
		{"auth.md", "Auth Package", "RBAC, ABAC, and capability-based security", mainPath("auth")},
		{"eval.md", "Eval Package", "Evaluation framework: metrics, runners, and provider integrations", func() []string {
			result := pathsWithPrefix("eval")
			return result
		}()},
		{"state.md", "State Package", "Shared agent state with watch and notify", func() []string {
			result := mainPath("state")
			result = append(result, providerPaths("state/")...)
			return result
		}()},
		{"prompt.md", "Prompt Package", "Prompt management, templating, and versioning", func() []string {
			result := mainPath("prompt")
			result = append(result, providerPaths("prompt/")...)
			return result
		}()},
		{"orchestration.md", "Orchestration Package", "Chain, Graph, Router, Parallel, and Supervisor orchestration patterns", mainPath("orchestration")},
		{"workflow.md", "Workflow Package", "Durable execution engine with provider integrations", func() []string {
			result := mainPath("workflow")
			result = append(result, providerPaths("workflow/")...)
			return result
		}()},

		// Protocol & Server
		{"protocol.md", "Protocol Package", "Protocol abstractions for MCP, A2A, REST, and OpenAI Agents compatibility", mainPath("protocol")},
		{"protocol-mcp.md", "MCP Protocol", "Model Context Protocol server/client, SDK, registry, and Composio integration", func() []string {
			var result []string
			for _, p := range allPaths {
				if strings.HasPrefix(p, "protocol/mcp") {
					result = append(result, p)
				}
			}
			return result
		}()},
		{"protocol-a2a.md", "A2A Protocol", "Agent-to-Agent protocol types and SDK implementation", func() []string {
			var result []string
			for _, p := range allPaths {
				if strings.HasPrefix(p, "protocol/a2a") {
					result = append(result, p)
				}
			}
			return result
		}()},
		{"protocol-rest.md", "REST & OpenAI Agents", "REST/SSE API server and OpenAI Agents protocol compatibility", func() []string {
			var result []string
			for _, p := range allPaths {
				if p == "protocol/rest" || p == "protocol/openai_agents" {
					result = append(result, p)
				}
			}
			return result
		}()},
		{"server.md", "Server Adapters", "HTTP framework adapters: Gin, Fiber, Echo, Chi, gRPC, Connect, Huma", func() []string {
			return pathsWithPrefix("server")
		}()},
		{"o11y.md", "Observability Package", "OpenTelemetry GenAI conventions, tracing, and provider integrations", func() []string {
			return pathsWithPrefix("o11y")
		}()},
	}

	for _, spec := range specs {
		if len(spec.paths) == 0 {
			continue
		}

		var groups []packageGroup
		for _, p := range spec.paths {
			if pg, ok := pkgs[p]; ok {
				groups = append(groups, pg)
			}
		}

		if len(groups) == 0 {
			continue
		}

		pages = append(pages, page{
			Filename:    spec.filename,
			Title:       spec.title,
			Description: spec.description,
			Packages:    groups,
		})
	}

	return pages
}

// renderPage generates Markdown content for a page.
func renderPage(p page) string {
	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: \"%s\"\n", escapeYAML(p.Title)))
	sb.WriteString(fmt.Sprintf("description: \"%s\"\n", escapeYAML(p.Description)))
	sb.WriteString("---\n\n")

	for i, pkg := range p.Packages {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		// For multi-package pages, show a section header per package
		if len(p.Packages) > 1 {
			if pkg.IsProvider {
				// Extract short provider name from import path
				parts := strings.Split(pkg.ImportPath, "/")
				provName := parts[len(parts)-1]
				sb.WriteString(fmt.Sprintf("## %s\n\n", provName))
			} else {
				sb.WriteString(fmt.Sprintf("## %s\n\n", pkg.PkgName))
			}
		}

		// Import path
		sb.WriteString(fmt.Sprintf("```go\nimport \"%s\"\n```\n\n", pkg.ImportPath))

		// Convert doc comment to markdown
		md := godocToMarkdown(pkg.DocComment)
		if md != "" {
			sb.WriteString(md)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderIndex generates the API reference index page.
func renderIndex(pages []page) string {
	var sb strings.Builder

	sb.WriteString(`---
title: "API Reference"
description: "Complete API documentation for all Beluga AI v2 packages, generated from source."
---

Complete API reference for all Beluga AI v2 packages. This documentation is generated from the Go source code doc comments.

## Foundation

| Package | Description |
|---------|-------------|
`)

	// Foundation
	foundationPages := []string{"core.md", "schema.md", "config.md"}
	for _, pg := range pages {
		for _, fp := range foundationPages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString("\n## LLM & Agents\n\n| Package | Description |\n|---------|-------------|\n")

	llmPages := []string{"llm.md", "llm-providers.md", "agent.md", "agent-workflow.md", "tool.md"}
	for _, pg := range pages {
		for _, fp := range llmPages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString("\n## Memory & RAG\n\n| Package | Description |\n|---------|-------------|\n")

	ragPages := []string{"memory.md", "memory-stores.md", "rag-embedding.md", "rag-embedding-providers.md", "rag-vectorstore.md", "rag-vectorstore-providers.md", "rag-retriever.md", "rag-loader.md", "rag-splitter.md"}
	for _, pg := range pages {
		for _, fp := range ragPages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString("\n## Voice\n\n| Package | Description |\n|---------|-------------|\n")

	voicePages := []string{"voice.md", "voice-stt.md", "voice-tts.md", "voice-s2s.md", "voice-transport.md", "voice-vad.md"}
	for _, pg := range pages {
		for _, fp := range voicePages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString("\n## Infrastructure\n\n| Package | Description |\n|---------|-------------|\n")

	infraPages := []string{"guard.md", "resilience.md", "cache.md", "hitl.md", "auth.md", "eval.md", "state.md", "prompt.md", "orchestration.md", "workflow.md"}
	for _, pg := range pages {
		for _, fp := range infraPages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString("\n## Protocol & Server\n\n| Package | Description |\n|---------|-------------|\n")

	protoPages := []string{"protocol.md", "protocol-mcp.md", "protocol-a2a.md", "protocol-rest.md", "server.md", "o11y.md"}
	for _, pg := range pages {
		for _, fp := range protoPages {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}

	sb.WriteString(`
## Design Patterns

All extensible packages in Beluga AI v2 follow consistent patterns:

### Registry Pattern

Every extensible package provides:
- ` + "`Register(name, factory)`" + ` — register providers in ` + "`init()`" + `
- ` + "`New(name, config)`" + ` — instantiate providers by name
- ` + "`List()`" + ` — discover available providers

### Middleware Pattern

Wrap interfaces to add cross-cutting behavior:
` + "```go\n" + `model = llm.ApplyMiddleware(model,
    llm.WithLogging(logger),
    llm.WithRetry(3),
)
` + "```\n" + `
### Hooks Pattern

Inject lifecycle callbacks without middleware:
` + "```go\n" + `hooks := llm.Hooks{
    BeforeGenerate: func(ctx, msgs) error { ... },
    AfterGenerate:  func(ctx, resp, err) { ... },
}
model = llm.WithHooks(model, hooks)
` + "```\n" + `
### Streaming Pattern

All streaming uses Go 1.23+ ` + "`iter.Seq2`" + `:
` + "```go\n" + `for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil { break }
    fmt.Print(chunk.Delta)
}
` + "```\n")

	return sb.String()
}

// escapeYAML escapes special characters in YAML string values.
func escapeYAML(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

