#!/usr/bin/env bash

set -e

REPO="fractalops/fli"
BINARY="fli"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s)
case "$OS" in
    Linux*)   OS=Linux;;
    Darwin*)  OS=Darwin;;
    MINGW*|MSYS*|CYGWIN*) OS=Windows;;
    *)        echo "Unsupported OS: $OS"; exit 1;;
esac

# Detect ARCH
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) ARCH=x86_64;;
    arm64|aarch64) ARCH=arm64;;
    *) echo "Unsupported architecture: $ARCH"; exit 1;;
esac

# Compose download URL
EXT="tar.gz"
if [ "$OS" = "Windows" ]; then
    EXT="zip"
fi

LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f 4)
if [ -z "$LATEST" ]; then
    echo "Could not fetch latest release tag."; exit 1
fi

ASSET="${BINARY}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/$REPO/releases/download/$LATEST/$ASSET"

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# Download
curl -LO "$URL"

# Extract
if [ "$EXT" = "tar.gz" ]; then
    tar -xzf "$ASSET"
else
    unzip -o "$ASSET"
fi

# Install
chmod +x "$BINARY"
mv "$BINARY" "$INSTALL_DIR/"

cd - >/dev/null
rm -rf "$TMP_DIR"

echo "fli installed to $INSTALL_DIR/$BINARY"
echo "Run 'fli --help' to get started!" 