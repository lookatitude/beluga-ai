# TODO: Prompts Package Redesign

## Current Issues
- The prompts package currently lacks configuration options and metrics.
- It needs improved templating capabilities.

## Required Changes
- Add `config.go` to handle prompt templates, including loading from files.
- Add factories for prompt creation to improve usability and extensibility.
- Integrate with schema for standardized message formats.
- Add metrics for prompt generations to enhance observability.
- Add tracing for template rendering processes.
- Enhance error handling for template failures with custom error types.
- Update tests to include configurations and new features.
- Document the package in `README.md` with usage examples.

## Goals
- Ensure the package is consistent with Beluga AI Framework principles.
- Make it easy to use, configure, and extend.
- Adhere to core principles: ISP, DIP, SRP, composition over inheritance.
- Include observability (OTEL traces/metrics, structured logging).
- Use proper configuration management with validation.
- Follow error handling best practices.

Note: Do not change the implementation yet; this is a planning document.
