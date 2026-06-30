#!/bin/sh
set -e

REPO="Koded0214h/ship-it"
BINARY="ship"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
DIM='\033[2m'
RESET='\033[0m'

info()  { printf "${DIM}  %s${RESET}\n" "$1"; }
ok()    { printf "${GREEN}  ✓ %s${RESET}\n" "$1"; }
fail()  { printf "${RED}  ✗ %s${RESET}\n" "$1"; exit 1; }

echo ""
echo "  Installing Ship..."
echo ""

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)   OS=linux ;;
  Darwin*)  OS=darwin ;;
  *)        fail "Unsupported OS: $OS" ;;
esac

# Detect arch
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)             fail "Unsupported architecture: $ARCH" ;;
esac

info "Detected: $OS/$ARCH"

# Fetch latest version from GitHub
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

if [ -z "$VERSION" ]; then
  fail "Could not determine latest version. Is the repo public and has a release?"
fi

info "Latest version: $VERSION"

# Build download URL
ARCHIVE="ship_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

info "Downloading $ARCHIVE"

# Download and extract
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/$ARCHIVE" || fail "Download failed: $URL"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  chmod +x "$INSTALL_DIR/$BINARY"
else
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
  sudo chmod +x "$INSTALL_DIR/$BINARY"
fi

ok "Installed to $INSTALL_DIR/$BINARY"
ok "ship $VERSION ready"
echo ""
echo "  Run: ship init"
echo ""
