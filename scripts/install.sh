#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

REPO="jhonxue/agent-fs"
BINARY_NAME="afs"
VERSION=${VERSION:-"latest"}

echo -e "${GREEN}Installing ${BINARY_NAME}...${NC}"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux*)
        OS="linux"
        ;;
    Darwin*)
        OS="darwin"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS="windows"
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Construct download URL and binary filename
DOWNLOAD_FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    DOWNLOAD_FILENAME="${DOWNLOAD_FILENAME}.exe"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/${VERSION}/download/${DOWNLOAD_FILENAME}"

echo -e "${YELLOW}OS: ${OS}${NC}"
echo -e "${YELLOW}Architecture: ${ARCH}${NC}"
echo -e "${YELLOW}Download URL: ${DOWNLOAD_URL}${NC}"
echo ""

# Determine install directory
INSTALL_DIR=""
if [ -d "$HOME/.local/bin" ] && echo ":$PATH:" | grep -q ":$HOME/.local/bin:"; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -d "$HOME/bin" ] && echo ":$PATH:" | grep -q ":$HOME/bin:"; then
    INSTALL_DIR="$HOME/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    echo -e "${YELLOW}Creating install directory: $INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR"

    # Add to PATH if not already present
    if ! echo ":$PATH:" | grep -q ":$INSTALL_DIR:"; then
        echo -e "${YELLOW}Adding $INSTALL_DIR to PATH${NC}"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.bashrc"
        echo -e "${YELLOW}Please run 'source ~/.bashrc' or restart your shell${NC}"
    fi
fi

# Local binary name (always afs, even on Windows)
BINARY_PATH="${INSTALL_DIR}/${BINARY_NAME}"

# Download binary
echo -e "${GREEN}Downloading ${BINARY_NAME}...${NC}"
if command -v wget >/dev/null 2>&1; then
    wget -q --show-progress -O "${BINARY_PATH}" "${DOWNLOAD_URL}"
elif command -v curl >/dev/null 2>&1; then
    curl -sL -o "${BINARY_PATH}" "${DOWNLOAD_URL}"
else
    echo -e "${RED}Neither wget nor curl is installed${NC}"
    exit 1
fi

# Make binary executable
chmod +x "${BINARY_PATH}"

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo -e "${GREEN}Binary installed to: ${BINARY_PATH}${NC}"
echo ""
echo -e "${YELLOW}Verify installation:${NC}"
echo "  ${BINARY_NAME} version"
echo ""
echo -e "${YELLOW}Quick start:${NC}"
echo "  ${BINARY_NAME} local info /path/to/file"
echo "  ${BINARY_NAME} local read /path/to/log --tail 50"
