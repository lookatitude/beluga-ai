# Data Model: Complete Documentation Overhaul

**Date**: 2025-01-27  
**Feature**: Complete Documentation Overhaul

## Entities

### Documentation Page

**Description**: A single documentation page with content, metadata, navigation links, and code examples.

**Attributes**:
- `title` (string, required): Page title
- `category` (string, required): Documentation category (getting-started, concepts, api, examples, guides, etc.)
- `framework_version` (string, required): Framework version this page applies to (e.g., "1.4.2")
- `content` (markdown, required): Page content in Markdown format
- `frontmatter` (object, optional): Docusaurus frontmatter (sidebar_position, id, etc.)
- `code_examples` (array, optional): Array of code examples embedded in the page
- `cross_references` (array, optional): Links to related documentation pages
- `last_updated` (date, optional): Last update date
- `deprecated` (boolean, default: false): Whether this page documents deprecated features

**Relationships**:
- Belongs to a `Documentation Section` (via category)
- Has many `Code Examples`
- References many `Documentation Pages` (via cross_references)
- Belongs to a `Framework Version`

**Validation Rules**:
- Title must be unique within category and version
- Framework version must match semantic versioning (MAJOR.MINOR.PATCH)
- Content must be valid Markdown
- All cross-references must point to existing pages

### Example

**Description**: A runnable code example with source code, description, prerequisites, usage instructions, and expected output.

**Attributes**:
- `name` (string, required): Example name/identifier
- `description` (string, required): What the example demonstrates
- `category` (string, required): Example category (agents, rag, voice, orchestration, etc.)
- `source_code` (string, required): Complete runnable source code
- `prerequisites` (array, optional): Required setup/dependencies
- `usage_instructions` (string, required): How to run the example
- `expected_output` (string, optional): Expected output when run
- `related_concepts` (array, optional): Related documentation pages
- `framework_version` (string, required): Framework version this example works with

**Relationships**:
- Belongs to a `Documentation Page` (example documentation page)
- References many `Documentation Pages` (via related_concepts)
- Belongs to a `Framework Version`

**Validation Rules**:
- Source code must be valid Go code
- Example must be runnable with specified prerequisites
- Framework version must match semantic versioning

### API Reference Entry

**Description**: Documentation for a single API function/type/interface with signature, parameters, return values, error conditions, usage examples, and cross-references.

**Attributes**:
- `package_name` (string, required): Go package name (e.g., "llms", "agents", "voice/stt")
- `entity_name` (string, required): Function/type/interface name
- `entity_type` (enum, required): Type of entity (function, type, interface, struct, constant)
- `signature` (string, required): Complete function/type signature
- `description` (string, required): What the entity does
- `parameters` (array, optional): Parameter descriptions (for functions)
- `return_values` (array, optional): Return value descriptions (for functions)
- `error_conditions` (array, optional): Possible errors and how to handle them
- `usage_examples` (array, optional): Code examples showing usage
- `related_entities` (array, optional): Related functions/types in same or other packages
- `framework_version` (string, required): Framework version this API applies to
- `deprecated` (boolean, default: false): Whether this API is deprecated
- `deprecation_notice` (string, optional): Deprecation notice and migration path

**Relationships**:
- Belongs to a `Package` (via package_name)
- References many `API Reference Entries` (via related_entities)
- Has many `Usage Examples`
- Belongs to a `Framework Version`

**Validation Rules**:
- Signature must be valid Go syntax
- All parameters must have descriptions
- All return values must have descriptions
- Deprecated entities must have deprecation_notice

### Tutorial Step

**Description**: A single step in a tutorial with instructions, code snippets, explanations, and expected outcomes.

**Attributes**:
- `tutorial_name` (string, required): Tutorial identifier
- `step_number` (integer, required): Step order (1, 2, 3, ...)
- `title` (string, required): Step title
- `instructions` (string, required): What the user should do
- `code_snippets` (array, optional): Code snippets for this step
- `explanations` (string, optional): Why this step is important
- `expected_outcome` (string, required): What should happen after completing this step
- `troubleshooting` (string, optional): Common issues and solutions

**Relationships**:
- Belongs to a `Tutorial` (via tutorial_name)
- May reference `Documentation Pages` for additional context

**Validation Rules**:
- Step numbers must be sequential within a tutorial
- Instructions must be clear and actionable
- Expected outcome must be verifiable

### Navigation Structure

**Description**: The website's navigation hierarchy that organizes documentation into logical sections and provides clear paths between related content.

**Attributes**:
- `section_name` (string, required): Navigation section name
- `section_type` (enum, required): Type of section (main, category, subcategory)
- `items` (array, required): Child navigation items (pages or sub-sections)
- `order` (integer, required): Display order
- `collapsed` (boolean, default: false): Whether section is collapsed by default
- `framework_version` (string, optional): Framework version this section applies to (if version-specific)

**Relationships**:
- Contains many `Documentation Pages`
- May contain other `Navigation Structures` (nested)
- Belongs to a `Framework Version` (if version-specific)

**Validation Rules**:
- Section names must be unique within parent
- Order must be positive integer
- All referenced pages must exist

### Framework Version

**Description**: A framework version that has its own documentation set.

**Attributes**:
- `version` (string, required): Semantic version (e.g., "1.4.2")
- `is_latest` (boolean, default: false): Whether this is the latest version
- `release_date` (date, optional): Release date
- `documentation_path` (string, required): Path to versioned documentation

**Relationships**:
- Has many `Documentation Pages`
- Has many `Examples`
- Has many `API Reference Entries`

**Validation Rules**:
- Version must follow semantic versioning
- Only one version can be marked as latest
- Documentation path must exist

## State Transitions

### Documentation Page Lifecycle

1. **Draft** → Content being written
2. **Review** → Content under review
3. **Published** → Content live on website
4. **Deprecated** → Content moved to legacy section
5. **Archived** → Content removed from active navigation

### Example Lifecycle

1. **Created** → Example code written
2. **Tested** → Example verified to run successfully
3. **Documented** → Documentation page created
4. **Published** → Available on website
5. **Updated** → Example updated for new framework version
6. **Deprecated** → Example no longer recommended

## Validation Rules Summary

- All framework versions must follow semantic versioning (MAJOR.MINOR.PATCH)
- All documentation pages must have valid Markdown content
- All code examples must be runnable with specified prerequisites
- All API reference entries must have complete signatures and descriptions
- All cross-references must point to existing pages
- Deprecated content must have migration paths
- Navigation structure must ensure 3-click access to any major feature
