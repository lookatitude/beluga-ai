---
name: review
description: Run Security Review on specified code. Requires 2 consecutive clean passes.
---

Run a security review on the specified code or recent changes.

## Workflow

1. **Security Reviewer** checks code against all security checklists.
2. If issues found → report with severity and remediation.
3. If clean → report as clean pass N/2.
4. Loop until 2 consecutive clean passes.

## Checklists Applied

- Input validation & injection prevention
- Authentication & authorization
- Cryptography & data protection
- Concurrency & resource safety
- Error handling & information disclosure
- Dependencies & supply chain
- Architecture compliance
