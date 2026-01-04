# Security Advisories

This document tracks known security vulnerabilities in Beluga AI Framework dependencies.

## Current Status

Last updated: 2025-01-XX

### ✅ Resolved Vulnerabilities

- **golang.org/x/crypto** (v0.43.0 → v0.45.0+)
  - GO-2025-4134: Unbounded memory consumption in golang.org/x/crypto/ssh
  - GO-2025-4135: Malformed constraint may cause denial of service in golang.org/x/crypto/ssh/agent
  - **Status**: Fixed by upgrading to v0.45.0+

### ⚠️ Known Vulnerabilities (No Fix Available)

The following vulnerabilities exist in `github.com/ollama/ollama` and currently have **no fixed version available**:

#### Affected Package
- **github.com/ollama/ollama** (v0.13.5)
  - All vulnerabilities listed below affect versions up to and including v0.13.5
  - **Fixed in**: N/A (no patch available yet)

#### Vulnerability Details

1. **GO-2025-3824**: Cross-Domain Token Exposure
   - **Severity**: High
   - **Description**: Ollama vulnerable to Cross-Domain Token Exposure
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3824

2. **GO-2025-3695**: Denial of Service (DoS) Attack
   - **Severity**: High
   - **Description**: Ollama Server Vulnerable to Denial of Service (DoS) Attack
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3695

3. **GO-2025-3689**: Divide by Zero Vulnerability
   - **Severity**: Medium
   - **Description**: Ollama Divide by Zero Vulnerability
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3689

4. **GO-2025-3582**: Denial of Service via Null Pointer Dereference
   - **Severity**: High
   - **Description**: Ollama Denial of Service (DoS) via Null Pointer Dereference
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3582

5. **GO-2025-3559**: Divide By Zero vulnerability
   - **Severity**: Medium
   - **Description**: Ollama Divide By Zero vulnerability
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3559

6. **GO-2025-3558**: Out-of-Bounds Read
   - **Severity**: High
   - **Description**: Ollama Allows Out-of-Bounds Read
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3558

7. **GO-2025-3557**: Resource Allocation Without Limits
   - **Severity**: High
   - **Description**: Ollama Allocation of Resources Without Limits or Throttling vulnerability
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3557

8. **GO-2025-3548**: Denial of Service via Crafted GZIP
   - **Severity**: High
   - **Description**: Ollama Vulnerable to Denial of Service (DoS) via Crafted GZIP
   - **More info**: https://pkg.go.dev/vuln/GO-2025-3548

#### Affected Code Paths

These vulnerabilities are reachable through:
- `pkg/embeddings/providers/ollama/ollama.go` - OllamaEmbedder.EmbedQuery
- `pkg/embeddings/providers/ollama/ollama.go` - NewOllamaEmbedder
- `pkg/prompts/iface/interfaces.go` - PromptError.Error (indirect)

## Recommendations

### For Production Deployments

1. **Monitor for Updates**: Regularly check for Ollama updates that address these vulnerabilities
   ```bash
   go list -m -u github.com/ollama/ollama
   ```

2. **Use Alternative Providers**: Consider using alternative embedding providers if Ollama is not required:
   - OpenAI embeddings (`pkg/embeddings/providers/openai`)
   - Other providers as they become available

3. **Network Isolation**: If using Ollama, ensure it's deployed in a network-isolated environment with proper access controls

4. **Input Validation**: Implement additional input validation and rate limiting when using Ollama embeddings

5. **Regular Scanning**: Run vulnerability scans regularly:
   ```bash
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   ```

### For Development

- These vulnerabilities primarily affect the Ollama client library used for embeddings
- Development environments should still exercise caution when testing with Ollama
- Consider using mock embeddings providers for testing when possible

## Reporting Security Issues

If you discover a security vulnerability in Beluga AI Framework itself (not in dependencies), please report it responsibly:

1. **Do NOT** open a public GitHub issue
2. Email security concerns to: [security contact email]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if available)

## Vulnerability Scanning

To scan for vulnerabilities in your project:

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan the entire project
govulncheck ./...

# Scan only the main packages (excluding examples)
govulncheck ./pkg/...

# Get verbose output with call traces
govulncheck -show verbose ./pkg/...
```

## References

- [Go Vulnerability Database](https://pkg.go.dev/vuln)
- [govulncheck Documentation](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [Ollama GitHub Repository](https://github.com/ollama/ollama)
