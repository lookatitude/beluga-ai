# Internal Package

This directory contains private implementation details for the retrievers package.

Files in this directory are not part of the public API and may change without notice.

## Guidelines

- Place internal utilities and helpers here
- Keep complex implementation details private
- Only expose interfaces and factory functions in the public API
- Use this directory for implementation-specific code that shouldn't be imported directly by users

## Current Contents

This directory is currently empty. As the package grows, implementation details such as:

- Complex retrieval algorithms
- Internal data structures
- Helper functions
- Private configuration structs

Should be placed here to maintain clean separation between public and private APIs.
