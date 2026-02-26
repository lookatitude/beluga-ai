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

// skippedDirs contains directory names to skip during doc collection.
var skippedDirs = map[string]bool{
	"vendor": true, ".git": true, "node_modules": true, "docs": true, "examples": true,
}

// collectDocFiles finds all doc.go files and extracts package doc comments.
func collectDocFiles(root string) (map[string]packageGroup, error) {
	pkgs := make(map[string]packageGroup)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		base := filepath.Base(path)
		if info.IsDir() && skippedDirs[base] {
			return filepath.SkipDir
		}
		if base != "doc.go" {
			return nil
		}

		pg, rel, ok := processDocFile(root, path)
		if ok {
			pkgs[rel] = pg
		}
		return nil
	})

	return pkgs, err
}

// processDocFile processes a single doc.go file and returns its packageGroup
// and relative path. Returns false if the file should be skipped.
func processDocFile(root, path string) (packageGroup, string, bool) {
	rel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return packageGroup{}, "", false
	}

	if strings.Contains(rel, "internal/") || rel == "internal" {
		return packageGroup{}, "", false
	}

	importPath := modulePath + "/" + filepath.ToSlash(rel)
	if rel == "." {
		importPath = modulePath
	}

	doc, pkgName, err := extractDocComment(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse %s: %v\n", path, err)
		return packageGroup{}, "", false
	}

	isProvider := strings.Contains(rel, "/providers/") || strings.Contains(rel, "/stores/") || strings.Contains(rel, "/adapters/")

	return packageGroup{
		ImportPath: importPath,
		PkgName:    pkgName,
		DocComment: doc,
		IsProvider: isProvider,
	}, rel, true
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
		var processed string
		processed, inCode = convertGodocLine(line, lines, i, inCode, &out)
		if processed != "" {
			out = append(out, processed)
		}
	}

	if inCode {
		out = append(out, "```")
	}

	result := strings.Join(out, "\n")
	result = multiBlankRe.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}

// convertGodocLine processes a single godoc line and appends to out as needed.
// Returns any final line to append and the updated inCode state.
func convertGodocLine(line string, lines []string, i int, inCode bool, out *[]string) (string, bool) {
	// Godoc section headers
	if strings.HasPrefix(line, "# ") {
		if inCode {
			*out = append(*out, "```")
			inCode = false
		}
		return "##" + line[1:], inCode
	}

	// Code block detection
	if isCodeLine(line) {
		if !inCode {
			*out = append(*out, "```go")
			inCode = true
		}
		return strings.TrimPrefix(line, "\t"), inCode
	}

	// End code block — but peek ahead for continuation
	if inCode && !isCodeLine(line) {
		if line == "" && i+1 < len(lines) && isCodeLine(lines[i+1]) {
			return "", inCode
		}
		*out = append(*out, "```")
		inCode = false
	}

	// List items
	if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "   - ") {
		return strings.TrimLeft(line, " "), inCode
	}

	// List continuation
	if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "    -") {
		return "  " + strings.TrimLeft(line, " "), inCode
	}

	return processGodocLinks(line), inCode
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

// pageSpec defines how to generate a single documentation page.
type pageSpec struct {
	filename    string
	title       string
	description string
	paths       []string
}

// pathResolver provides helper methods for resolving package paths.
type pathResolver struct {
	allPaths []string
	pkgs     map[string]packageGroup
}

// withPrefix returns all paths matching a prefix.
func (r *pathResolver) withPrefix(prefix string) []string {
	var result []string
	for _, p := range r.allPaths {
		if strings.HasPrefix(p, prefix) {
			result = append(result, p)
		}
	}
	return result
}

// providers returns provider/store/adapter paths under a prefix.
func (r *pathResolver) providers(prefix string) []string {
	var result []string
	for _, p := range r.allPaths {
		if strings.HasPrefix(p, prefix) && (strings.Contains(p, "/providers/") || strings.Contains(p, "/stores/") || strings.Contains(p, "/adapters/")) {
			result = append(result, p)
		}
	}
	return result
}

// main returns a single-element slice if the path exists, nil otherwise.
func (r *pathResolver) main(path string) []string {
	if _, ok := r.pkgs[path]; ok {
		return []string{path}
	}
	return nil
}

// mainAndProviders returns the main path plus provider paths under a prefix.
func (r *pathResolver) mainAndProviders(mainPkg, providerPrefix string) []string {
	result := r.main(mainPkg)
	result = append(result, r.providers(providerPrefix)...)
	return result
}

// buildPages groups packages into page definitions.
func buildPages(pkgs map[string]packageGroup) []page {
	allPaths := make([]string, 0, len(pkgs))
	for p := range pkgs {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	res := &pathResolver{allPaths: allPaths, pkgs: pkgs}
	specs := buildPageSpecs(res)
	return specsToPages(specs, pkgs)
}

// buildPageSpecs defines all documentation page specifications.
func buildPageSpecs(r *pathResolver) []pageSpec {
	return []pageSpec{
		// Foundation
		{"core.md", "Core Package", "Foundation primitives: streams, Runnable, events, errors, lifecycle, multi-tenancy", r.main("core")},
		{"schema.md", "Schema Package", "Shared types: messages, content parts, tool definitions, documents, events, sessions", r.main("schema")},
		{"config.md", "Config Package", "Configuration loading, validation, environment variables, and hot-reload", r.main("config")},
		// LLM
		{"llm.md", "LLM Package", "ChatModel interface, provider registry, middleware, hooks, structured output, routing", r.main("llm")},
		{"llm-providers.md", "LLM Providers", "All LLM provider implementations: OpenAI, Anthropic, Google, Ollama, Bedrock, and more", r.providers("llm/")},
		// Agent
		{"agent.md", "Agent Package", "Agent runtime, BaseAgent, Executor, Planner strategies, handoffs, and event bus", r.main("agent")},
		{"agent-workflow.md", "Agent Workflows", "Sequential, Parallel, and Loop workflow agents for multi-agent orchestration", r.main("agent/workflow")},
		// Tool
		{"tool.md", "Tool Package", "Tool interface, FuncTool, registry, MCP client integration, and middleware", r.main("tool")},
		// Memory
		{"memory.md", "Memory Package", "MemGPT-inspired 3-tier memory: Core, Recall, Archival, graph memory, composite", r.main("memory")},
		{"memory-stores.md", "Memory Store Providers", "Memory store implementations: in-memory, Redis, PostgreSQL, SQLite, MongoDB, Neo4j, Memgraph, Dragonfly", r.providers("memory/")},
		// RAG
		{"rag-embedding.md", "RAG Embedding", "Embedder interface for converting text to vector embeddings", r.main("rag/embedding")},
		{"rag-embedding-providers.md", "Embedding Providers", "Embedding provider implementations: OpenAI, Cohere, Google, Jina, Mistral, Ollama, Voyage, and more", r.providers("rag/embedding/")},
		{"rag-vectorstore.md", "RAG Vector Store", "VectorStore interface for similarity search over document embeddings", r.main("rag/vectorstore")},
		{"rag-vectorstore-providers.md", "Vector Store Providers", "Vector store implementations: pgvector, Pinecone, Qdrant, Weaviate, Milvus, Elasticsearch, and more", r.providers("rag/vectorstore/")},
		{"rag-retriever.md", "RAG Retriever", "Retriever strategies: Vector, Hybrid, HyDE, CRAG, Multi-Query, Ensemble, Rerank, Adaptive", r.main("rag/retriever")},
		{"rag-loader.md", "RAG Document Loaders", "Document loaders for files, cloud storage, APIs, and web content", r.mainAndProviders("rag/loader", "rag/loader/")},
		{"rag-splitter.md", "RAG Text Splitters", "Text splitting strategies for chunking documents", r.main("rag/splitter")},
		// Voice
		{"voice.md", "Voice Package", "Frame-based voice pipeline, VAD, hybrid cascade/S2S switching", r.main("voice")},
		{"voice-stt.md", "Voice STT", "Speech-to-text interface and providers: Deepgram, AssemblyAI, Whisper, Groq, ElevenLabs, Gladia", r.mainAndProviders("voice/stt", "voice/stt/")},
		{"voice-tts.md", "Voice TTS", "Text-to-speech interface and providers: ElevenLabs, Cartesia, PlayHT, Fish, Groq, LMNT, Smallest", r.mainAndProviders("voice/tts", "voice/tts/")},
		{"voice-s2s.md", "Voice S2S", "Speech-to-speech interface and providers: OpenAI Realtime, Gemini Live, Nova S2S", r.mainAndProviders("voice/s2s", "voice/s2s/")},
		{"voice-transport.md", "Voice Transport", "Transport layer for voice sessions: WebSocket, LiveKit, Daily, Pipecat", r.mainAndProviders("voice/transport", "voice/transport/")},
		{"voice-vad.md", "Voice VAD", "Voice activity detection providers: Silero, WebRTC", r.withPrefix("voice/vad/")},
		// Infrastructure
		{"guard.md", "Guard Package", "Three-stage safety pipeline: input, output, tool guards with built-in and external providers", r.mainAndProviders("guard", "guard/")},
		{"resilience.md", "Resilience Package", "Circuit breaker, hedge, retry, and rate limiting patterns", r.main("resilience")},
		{"cache.md", "Cache Package", "Exact, semantic, and prompt caching with pluggable backends", r.mainAndProviders("cache", "cache/")},
		{"hitl.md", "HITL Package", "Human-in-the-loop: confidence-based approval, escalation policies", r.main("hitl")},
		{"auth.md", "Auth Package", "RBAC, ABAC, and capability-based security", r.main("auth")},
		{"eval.md", "Eval Package", "Evaluation framework: metrics, runners, and provider integrations", r.withPrefix("eval")},
		{"state.md", "State Package", "Shared agent state with watch and notify", r.mainAndProviders("state", "state/")},
		{"prompt.md", "Prompt Package", "Prompt management, templating, and versioning", r.mainAndProviders("prompt", "prompt/")},
		{"orchestration.md", "Orchestration Package", "Chain, Graph, Router, Parallel, and Supervisor orchestration patterns", r.main("orchestration")},
		{"workflow.md", "Workflow Package", "Durable execution engine with provider integrations", r.mainAndProviders("workflow", "workflow/")},
		// Protocol & Server
		{"protocol.md", "Protocol Package", "Protocol abstractions for MCP, A2A, REST, and OpenAI Agents compatibility", r.main("protocol")},
		{"protocol-mcp.md", "MCP Protocol", "Model Context Protocol server/client, SDK, registry, and Composio integration", r.withPrefix("protocol/mcp")},
		{"protocol-a2a.md", "A2A Protocol", "Agent-to-Agent protocol types and SDK implementation", r.withPrefix("protocol/a2a")},
		{"protocol-rest.md", "REST & OpenAI Agents", "REST/SSE API server and OpenAI Agents protocol compatibility", filterPaths(r.allPaths, func(p string) bool { return p == "protocol/rest" || p == "protocol/openai_agents" })},
		{"server.md", "Server Adapters", "HTTP framework adapters: Gin, Fiber, Echo, Chi, gRPC, Connect, Huma", r.withPrefix("server")},
		{"o11y.md", "Observability Package", "OpenTelemetry GenAI conventions, tracing, and provider integrations", r.withPrefix("o11y")},
	}
}

// filterPaths returns paths matching a predicate.
func filterPaths(allPaths []string, pred func(string) bool) []string {
	var result []string
	for _, p := range allPaths {
		if pred(p) {
			result = append(result, p)
		}
	}
	return result
}

// specsToPages converts page specifications into page objects by resolving package groups.
func specsToPages(specs []pageSpec, pkgs map[string]packageGroup) []page {
	var pages []page
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

// indexSection defines a section header and the filenames to include.
type indexSection struct {
	heading   string
	filenames []string
}

// renderIndex generates the API reference index page.
func renderIndex(pages []page) string {
	var sb strings.Builder

	sb.WriteString(`---
title: "API Reference"
description: "Complete API documentation for all Beluga AI v2 packages, generated from source."
---

Complete API reference for all Beluga AI v2 packages. This documentation is generated from the Go source code doc comments.

`)

	sections := []indexSection{
		{"Foundation", []string{"core.md", "schema.md", "config.md"}},
		{"LLM & Agents", []string{"llm.md", "llm-providers.md", "agent.md", "agent-workflow.md", "tool.md"}},
		{"Memory & RAG", []string{"memory.md", "memory-stores.md", "rag-embedding.md", "rag-embedding-providers.md", "rag-vectorstore.md", "rag-vectorstore-providers.md", "rag-retriever.md", "rag-loader.md", "rag-splitter.md"}},
		{"Voice", []string{"voice.md", "voice-stt.md", "voice-tts.md", "voice-s2s.md", "voice-transport.md", "voice-vad.md"}},
		{"Infrastructure", []string{"guard.md", "resilience.md", "cache.md", "hitl.md", "auth.md", "eval.md", "state.md", "prompt.md", "orchestration.md", "workflow.md"}},
		{"Protocol & Server", []string{"protocol.md", "protocol-mcp.md", "protocol-a2a.md", "protocol-rest.md", "server.md", "o11y.md"}},
	}

	for _, section := range sections {
		writeIndexSection(&sb, section.heading, section.filenames, pages)
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

// writeIndexSection writes one section of the index page: a heading and a table of matching pages.
func writeIndexSection(sb *strings.Builder, heading string, filenames []string, pages []page) {
	sb.WriteString(fmt.Sprintf("## %s\n\n| Package | Description |\n|---------|-------------|\n", heading))
	for _, pg := range pages {
		for _, fp := range filenames {
			if pg.Filename == fp {
				slug := strings.TrimSuffix(pg.Filename, ".md")
				sb.WriteString(fmt.Sprintf("| [%s](./%s/) | %s |\n", pg.Title, slug, pg.Description))
			}
		}
	}
	sb.WriteString("\n")
}

// escapeYAML escapes special characters in YAML string values.
func escapeYAML(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

