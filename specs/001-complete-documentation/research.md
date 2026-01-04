# Research: Complete Documentation Overhaul

**Date**: 2025-01-27  
**Feature**: Complete Documentation Overhaul  
**Branch**: 001-complete-documentation

## Research Tasks

### 1. Docusaurus Versioning Strategy

**Task**: Research how to implement documentation versioning in Docusaurus 3

**Decision**: Use Docusaurus versioning plugin (`@docusaurus/plugin-content-docs` with versioning enabled) to maintain separate documentation sets per framework version (v1.4.2, v1.4.3, etc.)

**Rationale**: 
- Docusaurus has built-in support for documentation versioning
- Allows users to select their framework version and see appropriate documentation
- Maintains separate docs per version without duplication
- Supports version dropdown in navigation

**Alternatives Considered**:
- Manual version directories: Too much duplication and maintenance overhead
- Single docs with version tags: Confusing for users, hard to maintain
- External versioning system: Unnecessary complexity when Docusaurus supports it natively

**Implementation Notes**:
- Configure versioning in `docusaurus.config.js`
- Create versioned docs structure: `website/versioned_docs/version-1.4.2/`
- Set up version dropdown in navbar
- Default to latest version

### 2. Full-Text Search Implementation

**Task**: Research full-text search options for Docusaurus including code examples and comments

**Decision**: Use Algolia DocSearch (if available) or Docusaurus built-in search with enhanced indexing

**Rationale**:
- Docusaurus has built-in search that indexes all content including code blocks
- Can be enhanced with Algolia DocSearch for better search experience
- Supports indexing of markdown content, code examples, and comments
- Free tier available for open source projects

**Alternatives Considered**:
- Custom search implementation: Too complex, reinventing the wheel
- External search service: Additional dependency and cost
- Client-side only search: Limited functionality for large documentation sets

**Implementation Notes**:
- Configure search in `docusaurus.config.js`
- Ensure code blocks are indexed (default behavior)
- Test search with common queries (installation, voice agents, API reference)
- Consider Algolia DocSearch for production if needed

### 3. Deprecated Features Documentation Strategy

**Task**: Research best practices for documenting deprecated features in Docusaurus

**Decision**: Create separate "Legacy" or "Deprecated" section in sidebar with limited visibility, accessible but not prominently displayed

**Rationale**:
- Keeps deprecated content accessible for users who need it
- Doesn't clutter main navigation
- Follows common documentation patterns
- Can be linked from migration guides

**Alternatives Considered**:
- Remove deprecated docs entirely: Users on older versions need access
- Inline deprecation notices: Clutters main documentation
- Separate website: Too much overhead

**Implementation Notes**:
- Create `docs/deprecated/` or `docs/legacy/` directory
- Add to sidebar with collapsed/limited visibility
- Include deprecation notices with migration paths
- Link from relevant sections when features are deprecated

### 4. API Documentation Generation

**Task**: Research how to generate comprehensive API documentation from Go code

**Decision**: Use gomarkdoc tool to automatically extract godoc comments from Go code and generate markdown documentation for website

**Rationale**:
- Go's godoc system provides good base documentation embedded in code
- gomarkdoc automatically extracts godoc comments and converts to markdown
- Existing script `scripts/generate-docs.sh` already implements this process
- Automation ensures documentation stays in sync with code changes
- Manual enhancement can be added post-generation for examples and cross-references
- Balance between automation and quality

**Alternatives Considered**:
- Fully automated generation: Lacks context and examples (current approach allows post-generation enhancement)
- Fully manual: Too time-consuming and error-prone, documentation drifts from code
- External tools (pkg.go.dev): Doesn't match website design
- Standard godoc server: Not integrated with Docusaurus website

**Implementation Notes**:
- Use existing `scripts/generate-docs.sh` script that uses gomarkdoc
- Script generates markdown from godoc comments in all packages
- Output goes to `website/docs/api/packages/` directory
- Script handles: main packages, LLM providers, voice packages, tools
- Post-generation: Add examples, cross-references, and user-friendly enhancements
- Ensure all public functions have complete godoc comments in source code
- Verify script works correctly and handles all packages
- Integrate script into CI/CD to auto-update docs on code changes

### 5. Example Documentation Structure

**Task**: Research best practices for documenting code examples in documentation websites

**Decision**: Create example documentation pages with description, prerequisites, code, explanation, and expected output

**Rationale**:
- Examples need context to be useful
- Users need to understand what each example demonstrates
- Prerequisites help users run examples successfully
- Expected output helps verify correct execution

**Alternatives Considered**:
- Code-only examples: Too minimal, lacks context
- Overly verbose examples: Overwhelming for users
- Interactive examples: Complex to implement, not necessary for Go examples

**Implementation Notes**:
- Create `website/docs/examples/` section
- Document each example in `examples/` directory
- Include: description, prerequisites, code, step-by-step explanation, expected output
- Link examples from relevant concept/API pages

### 6. Navigation and Cross-Reference Strategy

**Task**: Research how to create effective navigation and cross-references in Docusaurus

**Decision**: Use Docusaurus sidebar configuration + markdown links for cross-references

**Rationale**:
- Docusaurus sidebar provides clear navigation structure
- Markdown links enable easy cross-referencing
- Can organize by user journey (getting started → concepts → API → examples)
- Supports deep linking and search

**Alternatives Considered**:
- Flat navigation: Confusing for users
- Complex nested navigation: Hard to maintain
- External navigation system: Unnecessary complexity

**Implementation Notes**:
- Configure `sidebars.js` with logical grouping
- Use relative markdown links for cross-references
- Ensure 3-click rule: any major feature within 3 clicks from homepage
- Test navigation paths for all user stories

### 7. Voice Agents Documentation Completeness

**Task**: Research what documentation is needed for Voice Agents feature

**Decision**: Document all 7 voice components (STT, TTS, VAD, Turn Detection, Transport, Noise Cancellation, Session Management) with usage examples, provider comparisons, and integration guides

**Rationale**:
- Voice Agents is a major feature in v1.4.2
- Users need comprehensive documentation to adopt it
- Each component needs individual documentation
- Integration examples show how components work together

**Alternatives Considered**:
- High-level overview only: Insufficient for implementation
- Component docs without examples: Hard to understand
- Examples without component docs: Missing context

**Implementation Notes**:
- Create `website/docs/voice/` section with sub-sections for each component
- Document each provider (Deepgram, Google, Azure, OpenAI, etc.)
- Include provider comparison guides
- Create integration examples showing all components together
- Link from main voice agents overview page

### 8. Godoc Automated Process Verification

**Task**: Ensure the godoc automated documentation generation process works properly and update website API reference

**Decision**: Verify and improve existing `scripts/generate-docs.sh` script that uses gomarkdoc to extract godoc comments and generate API documentation

**Rationale**:
- Existing automation script (`scripts/generate-docs.sh`) uses gomarkdoc to extract godoc from Go code
- Automation ensures API documentation stays synchronized with code changes
- Need to verify script works correctly for all packages (main packages, LLM providers, voice packages)
- Need to ensure generated markdown is properly formatted for Docusaurus/MDX
- Need to verify all public functions have proper godoc comments in source code
- Need to integrate into CI/CD pipeline for automatic updates

**Current State**:
- Script exists at `scripts/generate-docs.sh`
- Uses gomarkdoc tool to generate markdown from godoc comments
- Handles: main packages, LLM providers (anthropic, bedrock, cohere, ollama, openai), voice packages (stt, tts, vad, turndetection, transport, noise, session), tools
- Outputs to `website/docs/api/packages/` with proper frontmatter
- Includes MDX compatibility fixes (converts <details> tags, fixes escaped characters)

**Verification Needed**:
- Test script execution for all packages
- Verify generated markdown renders correctly in Docusaurus
- Check that all public functions have godoc comments
- Verify script handles edge cases (missing packages, empty docs, etc.)
- Test CI/CD integration for automatic doc updates
- Verify generated docs match current codebase

**Improvements Needed**:
- Add validation to check godoc comment completeness
- Add error handling for missing or incomplete godoc comments
- Ensure script handles all packages correctly (verify package list is complete)
- Add CI/CD integration to auto-generate docs on code changes
- Add verification step to ensure docs are up-to-date before merge

**Implementation Notes**:
- Run `scripts/generate-docs.sh` to regenerate all API documentation
- Verify generated files in `website/docs/api/packages/` are correct
- Check that all packages are included (compare script package list with actual packages)
- Test Docusaurus build with generated docs
- Add script to CI/CD pipeline (e.g., GitHub Actions workflow)
- Create validation script to check godoc comment completeness

## Summary

All research tasks completed. Key decisions:
1. Use Docusaurus built-in versioning for framework version documentation
2. Use Docusaurus search (with Algolia option) for full-text search including code
3. Create separate "Legacy" section for deprecated features
4. Use gomarkdoc automated extraction of godoc comments + manual enhancement for API documentation
5. Verify and improve existing godoc automation script to ensure it works properly
6. Create comprehensive example documentation with context
7. Use Docusaurus sidebar + markdown links for navigation
8. Document all 7 voice components comprehensively

All NEEDS CLARIFICATION items from spec have been resolved through clarifications session. Ready to proceed to Phase 1: Design & Contracts.
