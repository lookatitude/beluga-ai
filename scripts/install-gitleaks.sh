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

# Download URL (note: filename uses version without 'v' prefix)
VERSION_NUMBER=$(echo "$GITLEAKS_VERSION" | sed 's/^v//')
DOWNLOAD_URL="https://github.com/gitleaks/gitleaks/releases/download/${GITLEAKS_VERSION}/gitleaks_${VERSION_NUMBER}_${OS}_${ARCH}.tar.gz"

echo "Installing gitleaks ${GITLEAKS_VERSION} for ${OS}/${ARCH}..."
echo "Download URL: ${DOWNLOAD_URL}"

# Create temp directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download and extract
echo "Downloading gitleaks from $DOWNLOAD_URL..."
if command -v curl >/dev/null 2>&1; then
    if ! curl -sSL -f -o gitleaks.tar.gz "$DOWNLOAD_URL"; then
        echo "Error: Download failed"
        if [ -f gitleaks.tar.gz ]; then
            echo "Response received:"
            head -20 gitleaks.tar.gz
            rm -f gitleaks.tar.gz
        fi
        exit 1
    fi
elif command -v wget >/dev/null 2>&1; then
    if ! wget -q -O gitleaks.tar.gz "$DOWNLOAD_URL"; then
        echo "Error: Download failed"
        rm -f gitleaks.tar.gz
        exit 1
    fi
else
    echo "Error: Neither curl nor wget is available"
    exit 1
fi

# Check if file was downloaded and has content
if [ ! -f gitleaks.tar.gz ] || [ ! -s gitleaks.tar.gz ]; then
    echo "Error: Download failed - file is empty or missing"
    exit 1
fi

# Verify the downloaded file is actually a gzip archive
if ! file gitleaks.tar.gz | grep -q "gzip\|compressed"; then
    echo "Error: Downloaded file is not a valid gzip archive"
    echo "File type: $(file gitleaks.tar.gz)"
    echo "First 100 bytes:"
    head -c 100 gitleaks.tar.gz
    rm -f gitleaks.tar.gz
    exit 1
fi

echo "Extracting gitleaks..."
tar -xzf gitleaks.tar.gz || {
    echo "Error: Failed to extract archive"
    rm -f gitleaks.tar.gz
    exit 1
}

if [ ! -f "$BINARY_NAME" ]; then
    echo "Error: Binary $BINARY_NAME not found after extraction"
    ls -la
    exit 1
fi

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

