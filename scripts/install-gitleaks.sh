#!/bin/bash
# Install gitleaks for secret detection
# Supports Linux (x64), macOS (x64/arm64), and Windows

set -e

GITLEAKS_VERSION="v8.18.0"
OS=""
ARCH=""
BINARY_NAME="gitleaks"

# Detect OS
case "$(uname -s)" in
    Linux*)     OS="linux";;
    Darwin*)    OS="darwin";;
    MINGW*)     OS="windows"; BINARY_NAME="gitleaks.exe";;
    *)          echo "Unsupported OS: $(uname -s)"; exit 1;;
esac

# Detect architecture
case "$(uname -m)" in
    x86_64)     ARCH="x64";;
    arm64|aarch64) ARCH="arm64";;
    *)          echo "Unsupported architecture: $(uname -m)"; exit 1;;
esac

# For macOS, prefer arm64 if available, fallback to x64
if [ "$OS" = "darwin" ] && [ "$ARCH" = "x64" ]; then
    # Check if we're on Apple Silicon
    if sysctl -n machdep.cpu.brand_string | grep -q "Apple"; then
        ARCH="arm64"
    fi
fi

# Download URL
DOWNLOAD_URL="https://github.com/gitleaks/gitleaks/releases/download/${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_${OS}_${ARCH}.tar.gz"

echo "Installing gitleaks ${GITLEAKS_VERSION} for ${OS}/${ARCH}..."
echo "Download URL: ${DOWNLOAD_URL}"

# Create temp directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
if command -v curl >/dev/null 2>&1; then
    curl -sSL -o gitleaks.tar.gz "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -q -O gitleaks.tar.gz "$DOWNLOAD_URL"
else
    echo "Error: Neither curl nor wget is available"
    exit 1
fi

tar -xzf gitleaks.tar.gz
chmod +x "$BINARY_NAME"

# Install to a location in PATH
INSTALL_DIR=""
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -w "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
elif [ -w "$(go env GOPATH)/bin" ]; then
    INSTALL_DIR="$(go env GOPATH)/bin"
    mkdir -p "$INSTALL_DIR"
else
    echo "Error: Cannot find a writable directory in PATH"
    echo "Please install gitleaks manually:"
    echo "  sudo mv $TMP_DIR/$BINARY_NAME /usr/local/bin/"
    exit 1
fi

mv "$BINARY_NAME" "$INSTALL_DIR/gitleaks"
echo "✅ gitleaks installed to $INSTALL_DIR/gitleaks"

# Cleanup
cd -
rm -rf "$TMP_DIR"

# Verify installation
if command -v gitleaks >/dev/null 2>&1; then
    echo "✅ gitleaks is now available: $(which gitleaks)"
    gitleaks version
else
    echo "⚠️  gitleaks installed but not in PATH. Please add $INSTALL_DIR to your PATH"
fi

