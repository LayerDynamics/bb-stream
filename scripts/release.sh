#!/bin/bash

# Release script for BB-Stream
# Builds all binaries and creates release artifacts

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Get version from tauri.conf.json
VERSION=$(grep '"version"' "$PROJECT_ROOT/desktop/src-tauri/tauri.conf.json" | head -1 | sed 's/.*"\([0-9.]*\)".*/\1/')

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  BB-Stream Release Build v$VERSION${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Run tests
echo -e "${YELLOW}Step 1: Running tests...${NC}"
cd "$PROJECT_ROOT"
go test ./... || { echo -e "${RED}Tests failed!${NC}"; exit 1; }
echo -e "${GREEN}✓ Tests passed${NC}"
echo ""

# Step 2: Build Go sidecars
echo -e "${YELLOW}Step 2: Building Go sidecars...${NC}"
"$SCRIPT_DIR/build-sidecar.sh" --universal
echo ""

# Step 3: Check Svelte
echo -e "${YELLOW}Step 3: Running Svelte type check...${NC}"
cd "$PROJECT_ROOT/desktop"
npm run check || { echo -e "${RED}Svelte check failed!${NC}"; exit 1; }
echo -e "${GREEN}✓ Svelte check passed${NC}"
echo ""

# Step 4: Build Tauri app
echo -e "${YELLOW}Step 4: Building Tauri app...${NC}"
npm run tauri build
echo ""

# Step 5: Show artifacts
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Release Build Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Build artifacts:"
echo ""

# List the built bundles
BUNDLE_DIR="$PROJECT_ROOT/desktop/src-tauri/target/release/bundle"
if [ -d "$BUNDLE_DIR" ]; then
    echo "macOS:"
    ls -lh "$BUNDLE_DIR/dmg/"*.dmg 2>/dev/null || echo "  No DMG found"
    ls -lh "$BUNDLE_DIR/macos/"*.app 2>/dev/null || echo "  No .app found"
    echo ""
    echo "Windows:"
    ls -lh "$BUNDLE_DIR/msi/"*.msi 2>/dev/null || echo "  No MSI found"
    ls -lh "$BUNDLE_DIR/nsis/"*.exe 2>/dev/null || echo "  No NSIS installer found"
    echo ""
    echo "Linux:"
    ls -lh "$BUNDLE_DIR/appimage/"*.AppImage 2>/dev/null || echo "  No AppImage found"
    ls -lh "$BUNDLE_DIR/deb/"*.deb 2>/dev/null || echo "  No deb found"
fi

echo ""
echo -e "${GREEN}Done! Version $VERSION is ready for release.${NC}"
