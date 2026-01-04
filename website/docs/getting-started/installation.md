---
title: Installation
sidebar_position: 1
---

# Beluga AI Framework - Installation Guide

This guide provides comprehensive installation instructions for the Beluga AI Framework across different platforms and environments.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Quick Installation](#quick-installation)
3. [Platform-Specific Installation](#platform-specific-installation)
4. [Dependency Management](#dependency-management)
5. [Verification](#verification)
6. [Docker Installation](#docker-installation)
7. [Development Environment Setup](#development-environment-setup)
8. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

- **Go**: Version 1.24 or later
- **Operating System**: Linux, macOS, or Windows (including WSL)
- **Memory**: 2GB RAM minimum (4GB+ recommended)
- **Disk Space**: 500MB for framework and dependencies

### Optional Dependencies

Depending on which providers you use, you may need:

- **PostgreSQL** (for pgvector): Version 12+ with pgvector extension
- **Docker** (for containerized deployment): Version 20.10+
- **API Keys** for external providers:
  - OpenAI API key
  - Anthropic API key
  - AWS credentials (for Bedrock)
  - Pinecone API key (for vector store)

## Quick Installation

### For End Users

If you're using Beluga AI in your project:

```bash
go get github.com/lookatitude/beluga-ai
```

Or in a new project:

```bash
mkdir my-beluga-app
cd my-beluga-app
go mod init my-beluga-app
go get github.com/lookatitude/beluga-ai
```

### For Developers

If you're contributing to or developing with the framework:

```bash
# Clone the repository
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai

# Install dependencies
go mod download
go mod verify

# Install development tools
make install-tools

# Verify installation
make test
```

## Platform-Specific Installation

### Linux

#### Ubuntu/Debian

```bash
# Install Go 1.24 or later (if not already installed)
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

#### RHEL/CentOS/Fedora

```bash
# Install Go using package manager (if available)
sudo dnf install golang  # Fedora
# OR
sudo yum install golang  # RHEL/CentOS

# Or install from official binaries
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz

# Add to PATH
export PATH=$PATH:/usr/local/go/bin

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

#### Arch Linux

```bash
# Install Go from AUR or official repository
sudo pacman -S go

# Or install from official binaries
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz

# Add to PATH
export PATH=$PATH:/usr/local/go/bin

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

### macOS

#### Using Homebrew (Recommended)

```bash
# Install Go
brew install go

# Verify installation
go version

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

#### Manual Installation

```bash
# Download Go installer
# Visit https://go.dev/dl/ and download the macOS installer

# Or use curl
curl -O https://go.dev/dl/go1.24.2.darwin-amd64.pkg

# Install the package, then verify
go version

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

### Windows

#### Using Chocolatey

```bash
# Install Go
choco install golang

# Verify installation
go version

# Install Beluga AI
go get github.com/lookatitude/beluga-ai
```

#### Manual Installation

1. **Download Go installer:**
   - Visit https://go.dev/dl/
   - Download the Windows installer (`.msi` file)

2. **Run the installer:**
   - Follow the installation wizard
   - Go will be installed to `C:\Program Files\Go` by default

3. **Verify installation:**
   ```cmd
   go version
   ```

4. **Install Beluga AI:**
   ```cmd
   go get github.com/lookatitude/beluga-ai
   ```

#### Windows Subsystem for Linux (WSL)

If using WSL, follow the Linux installation instructions for your WSL distribution.

## Dependency Management

### Go Modules

Beluga AI uses Go modules for dependency management. The framework automatically manages its dependencies.

### External Dependencies

#### PostgreSQL with pgvector (Optional)

If you plan to use the PgVector vector store provider:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib

# Install pgvector extension
sudo apt-get install postgresql-14-pgvector  # Adjust version as needed

# Or compile from source
git clone --branch v0.5.1 https://github.com/pgvector/pgvector.git
cd pgvector
make
sudo make install
```

#### Docker (Optional)

For containerized deployments:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install docker.io docker-compose

# macOS
brew install docker docker-compose

# Windows
# Download Docker Desktop from https://www.docker.com/products/docker-desktop
```

### Provider-Specific Requirements

#### OpenAI Provider

- API key from https://platform.openai.com/api-keys
- No additional software required

#### Anthropic Provider

- API key from https://console.anthropic.com/
- No additional software required

#### AWS Bedrock Provider

- AWS account with Bedrock access
- AWS CLI configured (optional, for local development)
- IAM credentials with Bedrock permissions

#### Ollama Provider

- Ollama installed locally: https://ollama.ai
- Models downloaded via `ollama pull <model-name>`

#### Pinecone Provider

- Pinecone account and API key
- No additional software required

## Verification

### Check Go Version

```bash
go version
# Should output: go version go1.24.x or later
```

### Verify Module Installation

```bash
# In your project directory
go mod verify

# Check installed packages
go list -m all | grep beluga-ai
```

### Test Installation

Create a simple test file `test_installation.go`:

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
    ctx := context.Background()

    // Test with a simple configuration
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    factory := llms.NewFactory()
    provider, err := factory.CreateProvider("openai", config)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    messages := []schema.Message{
        schema.NewHumanMessage("Hello, world!"),
    }

    response, err := provider.Generate(ctx, messages)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Installation verified! Response: %s\n", response.Content)
}
```

Run the test:

```bash
export OPENAI_API_KEY="your-api-key-here"
go run test_installation.go
```

## Docker Installation

### Development Environment

Create a `Dockerfile.dev`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN make build

# Development image
FROM golang:1.24-alpine

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache git ca-certificates

# Copy binaries from builder
COPY --from=builder /app/bin /app/bin

# Copy source for development
COPY . .

CMD ["go", "run", "main.go"]
```

### Production Environment

Create a `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Final image
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/app .

# Run
CMD ["./app"]
```

### Docker Compose

Create a `docker-compose.yml` for development:

```yaml
version: '3.8'

services:
  beluga-app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    volumes:
      - .:/app
      - go-mod-cache:/go/pkg/mod
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    ports:
      - "8080:8080"
    depends_on:
      - postgres

  postgres:
    image: pgvector/pgvector:pg15
    environment:
      POSTGRES_USER: beluga
      POSTGRES_PASSWORD: beluga
      POSTGRES_DB: beluga
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  go-mod-cache:
  postgres-data:
```

## Development Environment Setup

### IDE Configuration

#### VS Code

1. **Install Go extension:**
   - Open VS Code
   - Go to Extensions (Ctrl+Shift+X / Cmd+Shift+X)
   - Search for "Go" by Google
   - Install the extension

2. **Configure settings** (`.vscode/settings.json`):
   ```json
   {
     "go.toolsManagement.checkForUpdates": "local",
     "go.useLanguageServer": true,
     "go.lintTool": "golangci-lint",
     "go.lintOnSave": "workspace",
     "go.formatTool": "gofumpt",
     "go.testFlags": ["-v", "-race"],
     "[go]": {
       "editor.formatOnSave": true,
       "editor.codeActionsOnSave": {
         "source.organizeImports": true
       }
     }
   }
   ```

3. **Install Go tools:**
   - Open Command Palette (Ctrl+Shift+P / Cmd+Shift+P)
   - Run "Go: Install/Update Tools"
   - Select all tools

#### GoLand

1. **Open project:**
   - File → Open → Select project directory

2. **Configure Go SDK:**
   - File → Settings → Go → GOROOT
   - Ensure Go 1.24 or later is selected

3. **Configure code style:**
   - File → Settings → Editor → Code Style → Go
   - Import from gofmt/gofumpt

4. **Enable inspections:**
   - File → Settings → Editor → Inspections → Go
   - Enable all relevant inspections

### Editor Plugins and Extensions

#### VS Code Extensions

- **Go** (by Google) - Official Go extension
- **golangci-lint** - Linting integration
- **Go Test** - Test runner
- **YAML** - For configuration files

#### Vim/Neovim

```vim
" Using vim-plug
Plug 'fatih/vim-go'
Plug 'buoto/gotests-vim'
```

#### Emacs

```elisp
;; Using use-package
(use-package go-mode
  :ensure t
  :config
  (add-hook 'go-mode-hook 'lsp-deferred))
```

### Debugging Setup

#### VS Code

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}",
      "env": {
        "OPENAI_API_KEY": "${env:OPENAI_API_KEY}"
      }
    },
    {
      "name": "Test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}"
    }
  ]
}
```

#### GoLand

1. Create run configuration:
   - Run → Edit Configurations
   - Click "+" → Go Build
   - Set program arguments and environment variables

2. Debug configuration:
   - Set breakpoints
   - Run → Debug (Shift+F9)

## Troubleshooting

### Common Installation Issues

#### Go Version Mismatch

**Problem:** `go version` shows version less than 1.24

**Solution:**
```bash
# Check current version
go version

# Update Go (see platform-specific instructions above)
# Verify after update
go version
```

#### Module Download Errors

**Problem:** `go mod download` fails with network errors

**Solution:**
```bash
# Set Go proxy (if behind firewall)
go env -w GOPROXY=https://proxy.golang.org,direct

# Or use direct mode
go env -w GOPROXY=direct

# Clear module cache
go clean -modcache

# Retry download
go mod download
```

#### Permission Errors

**Problem:** Permission denied when installing

**Solution:**
```bash
# Linux/macOS: Check GOROOT and GOPATH permissions
ls -la $GOROOT
ls -la $GOPATH

# Windows: Run terminal as Administrator if needed
```

#### Missing Dependencies

**Problem:** Build fails with missing package errors

**Solution:**
```bash
# Download all dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy module
go mod tidy
```

### Provider-Specific Issues

#### PostgreSQL/pgvector Issues

**Problem:** Cannot connect to PostgreSQL

**Solution:**
```bash
# Check PostgreSQL is running
sudo systemctl status postgresql  # Linux
brew services list | grep postgres  # macOS

# Verify pgvector extension
psql -U postgres -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

#### API Key Issues

**Problem:** Provider authentication fails

**Solution:**
```bash
# Verify environment variables are set
echo $OPENAI_API_KEY
echo $ANTHROPIC_API_KEY

# Test API key
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

### Getting Help

If you encounter issues not covered here:

1. **Check existing documentation:**
   - [Quick Start Guide](./quickstart)
   - [Troubleshooting Guide](../guides/troubleshooting)

2. **Search GitHub Issues:**
   - https://github.com/lookatitude/beluga-ai/issues

3. **Create a new issue:**
   - Include Go version (`go version`)
   - Include OS and version
   - Include error messages
   - Include steps to reproduce

## Next Steps

After installation, continue with:

1. **[Quick Start Guide](./quickstart)** - Get started in minutes
2. **[Getting Started Tutorial](./tutorials/first-llm-call)** - Step-by-step tutorials
3. **[Architecture Documentation](../guides/architecture)** - Understand the framework
4. **[Use Cases](../use-cases/)** - Real-world examples

## API Reference

As you work with Beluga AI, you'll interact with several core packages. Refer to the [API Reference](../api/index) for detailed documentation:

- **[Core Package](../api/packages/core)** - Core components, error handling, and utilities
- **[Config Package](../api/packages/config)** - Configuration management
- **[Schema Package](../api/packages/schema)** - Data schemas and type definitions

---

**Last Updated:** Installation guide is actively maintained. Check back for updates on new requirements or installation methods.

