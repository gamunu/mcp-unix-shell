#!/bin/bash

# MCP Unix Shell Install Script
# This script installs the latest version of mcp-unix-shell

set -e

# Default version - will be updated by CI/CD
VERSION="0.1.0"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Determine download URL
DOWNLOAD_URL="https://github.com/gamunu/mcp-unix-shell/releases/download/v${VERSION}/mcp-unix-shell_${VERSION}_${OS}_${ARCH}"

# Determine install location
INSTALL_DIR="$HOME/.local/bin"
if [ -d "$HOME/bin" ]; then
  INSTALL_DIR="$HOME/bin"
fi

# Create directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Download the binary
echo "Downloading mcp-unix-shell v${VERSION} for ${OS}_${ARCH}..."
curl -L -o "$INSTALL_DIR/mcp-unix-shell" "$DOWNLOAD_URL"
chmod +x "$INSTALL_DIR/mcp-unix-shell"

echo "Successfully installed mcp-unix-shell to $INSTALL_DIR/mcp-unix-shell"

# Check if installation directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo ""
  echo "WARNING: $INSTALL_DIR is not in your PATH."
  echo "Add the following line to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "To use with Claude Desktop, add the following to your claude_desktop_config.json:"
echo ""
echo "{
  \"mcpServers\": {
    \"shell\": {
      \"command\": \"mcp-unix-shell\",
      \"args\": [
        \"--allowed-commands=ls,cat,echo,find\"
      ]
    }
  }
}"
echo ""
echo "For more information, visit: https://github.com/gamunu/mcp-unix-shell"
