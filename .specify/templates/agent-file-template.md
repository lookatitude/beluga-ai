# [PROJECT NAME] Development Guidelines

Auto-generated from all feature plans. Last updated: [DATE]

## Constitutional Requirements (MANDATORY)
**ALL development MUST comply with Constitution v1.0.0**

### Package Structure (ENFORCED)
- REQUIRED: iface/, config.go, metrics.go, errors.go, test_utils.go, advanced_test.go
- MUST: Follow ISP (small interfaces), DIP (dependency injection), SRP (single responsibility)
- MUST: Use global registry pattern for multi-provider packages
- MUST: Implement OTEL metrics (NO custom metrics)
- MUST: Op/Err/Code error handling pattern

### Testing Standards (NON-NEGOTIABLE)  
- MUST: 100% test coverage with advanced mocks
- MUST: Table-driven tests, concurrency tests, benchmarks
- MUST: Integration tests for cross-package interactions

## Active Technologies
[EXTRACTED FROM ALL PLAN.MD FILES]

## Project Structure
```
[ACTUAL STRUCTURE FROM PLANS]
```

## Commands
[ONLY COMMANDS FOR ACTIVE TECHNOLOGIES]

## Code Style
[LANGUAGE-SPECIFIC, ONLY FOR LANGUAGES IN USE]

## Recent Changes
[LAST 3 FEATURES AND WHAT THEY ADDED]

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->