# References

## Prior Specifications

### 2026-01-26 Package Design Refactor
- **Location**: `agent-os/specs/2026-01-26-1500-package-design-refactor/`
- **Purpose**: Initial establishment of package design patterns
- **Key outcomes**: Defined iface/, providers/, registry patterns

### 2026-01-27 Package Standardization
- **Location**: `agent-os/specs/2026-01-27-pkg-standardization/`
- **Purpose**: Framework-wide standardization effort
- **Key outcomes**: All packages updated to follow patterns

### 2026-01-27 Package Flattening
- **Location**: `agent-os/specs/2026-01-27-pkg-flattening/`
- **Purpose**: Simplified package hierarchy
- **Key outcomes**: Voice packages restructured

### 2026-01-27 LLM Package Standardization
- **Location**: `agent-os/specs/2026-01-27-1200-llm-package-standardization/`
- **Purpose**: LLM-specific standardization
- **Key outcomes**: llms package as reference implementation

## Documentation References

### Package Design Patterns
- **Location**: `docs/package_design_patterns.md`
- **Purpose**: Canonical reference for all package conventions
- **Status**: Authoritative source for package structure rules

### CLAUDE.md
- **Location**: `CLAUDE.md`
- **Purpose**: AI assistant guidance
- **Relevant sections**: Package Structure Convention, Repository Structure

## Code References

### Registry Pattern Examples
- **embeddings/registry.go**: Reference implementation for registry pattern
- **llms/registry.go**: Additional reference implementation
- **memory/registry.go**: Memory-specific registry example

### Main File Examples
- **embeddings/embeddings.go**: Main API file example
- **llms/llms.go**: Main API with factory functions
- **agents/agents.go**: Main API with composite patterns
