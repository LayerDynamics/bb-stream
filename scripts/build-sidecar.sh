#!/bin/bash

# Build script for bb-stream sidecar binary
# Compiles Go binary for all platforms supported by Tauri

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/desktop/src-tauri/binaries"

# Binary name
BINARY_NAME="bb-stream"

# Build flags
LDFLAGS="-s -w"

# Parse arguments
BUILD_UNIVERSAL=false
BUILD_CURRENT=false
for arg in "$@"; do
    case $arg in
        --universal)
            BUILD_UNIVERSAL=true
            ;;
        --current)
            BUILD_CURRENT=true
            ;;
    esac
done

echo -e "${GREEN}Building bb-stream sidecar...${NC}"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

build_binary() {
    local goos=$1
    local goarch=$2
    local tauri_target=$3

    local output_name="$BINARY_NAME-$tauri_target"
    if [ "$goos" = "windows" ]; then
        output_name="$output_name.exe"
    fi

    local output_path="$OUTPUT_DIR/$output_name"

    echo -e "${YELLOW}Building for $goos/$goarch ($tauri_target)...${NC}"

    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build \
        -ldflags="$LDFLAGS" \
        -o "$output_path" \
        "$PROJECT_ROOT/cmd/bb-stream/"

    if [ $? -eq 0 ]; then
        local size=$(ls -lh "$output_path" | awk '{print $5}')
        echo -e "${GREEN}  ✓ Built $output_name ($size)${NC}"
    else
        echo -e "${RED}  ✗ Failed to build for $goos/$goarch${NC}"
        exit 1
    fi
}

build_universal_macos() {
    echo -e "${YELLOW}Building macOS universal binary...${NC}"

    local arm64_path="$OUTPUT_DIR/$BINARY_NAME-aarch64-apple-darwin"
    local x86_path="$OUTPUT_DIR/$BINARY_NAME-x86_64-apple-darwin"
    local universal_path="$OUTPUT_DIR/$BINARY_NAME-universal-apple-darwin"

    # Build both architectures
    build_binary "darwin" "arm64" "aarch64-apple-darwin"
    build_binary "darwin" "amd64" "x86_64-apple-darwin"

    # Create universal binary using lipo
    if command -v lipo &> /dev/null; then
        lipo -create -output "$universal_path" "$arm64_path" "$x86_path"
        local size=$(ls -lh "$universal_path" | awk '{print $5}')
        echo -e "${GREEN}  ✓ Created universal binary ($size)${NC}"
    else
        echo -e "${YELLOW}  ⚠ lipo not available, skipping universal binary${NC}"
    fi
}

if [ "$BUILD_CURRENT" = true ]; then
    # Build only for current platform
    CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    CURRENT_ARCH=$(uname -m)

    case "$CURRENT_OS" in
        darwin)
            if [ "$CURRENT_ARCH" = "arm64" ]; then
                build_binary "darwin" "arm64" "aarch64-apple-darwin"
            else
                build_binary "darwin" "amd64" "x86_64-apple-darwin"
            fi
            ;;
        linux)
            build_binary "linux" "amd64" "x86_64-unknown-linux-gnu"
            ;;
        *)
            echo -e "${RED}Unsupported platform: $CURRENT_OS${NC}"
            exit 1
            ;;
    esac
else
    # Build for all platforms

    # macOS
    if [ "$BUILD_UNIVERSAL" = true ]; then
        build_universal_macos
    else
        build_binary "darwin" "arm64" "aarch64-apple-darwin"
        build_binary "darwin" "amd64" "x86_64-apple-darwin"
    fi

    # Linux
    build_binary "linux" "amd64" "x86_64-unknown-linux-gnu"

    # Windows
    build_binary "windows" "amd64" "x86_64-pc-windows-msvc"
fi

echo ""
echo -e "${GREEN}Build completed successfully!${NC}"
echo ""
echo "Built binaries:"
ls -lh "$OUTPUT_DIR"
