#!/bin/bash
# Build release binaries for multiple platforms

set -e

# Configuration
BINARY_NAME="gofer"
MAIN_PATH="./cmd/gofer"
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"
BUILD_DIR="dist"
LDFLAGS="-X main.Version=${VERSION} -s -w"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Platforms to build for
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "windows/amd64"
    "windows/386"
)

# Print colored message
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Clean previous builds
print_message $BLUE "Cleaning previous builds..."
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a platform_split <<< "$platform"
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name="${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    print_message $BLUE "Building for ${GOOS}/${GOARCH}..."
    
    # Build binary
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
        go build -ldflags="${LDFLAGS}" \
        -o "${BUILD_DIR}/${output_name}" \
        ${MAIN_PATH}
    
    if [ $? -eq 0 ]; then
        print_message $GREEN "✓ Built ${output_name}"
        
        # Create archive
        cd ${BUILD_DIR}
        if [ "$GOOS" = "windows" ]; then
            # Create zip for Windows
            archive_name="${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.zip"
            zip -q "${archive_name}" "${output_name}"
            rm "${output_name}"
        else
            # Create tar.gz for Unix-like systems
            archive_name="${BINARY_NAME}-${VERSION}-${GOOS}-${GOARCH}.tar.gz"
            tar -czf "${archive_name}" "${output_name}"
            rm "${output_name}"
        fi
        cd ..
        
        print_message $GREEN "✓ Created ${archive_name}"
    else
        print_message $RED "✗ Failed to build for ${GOOS}/${GOARCH}"
    fi
done

# Generate checksums
print_message $BLUE "Generating checksums..."
cd ${BUILD_DIR}
shasum -a 256 *.tar.gz *.zip > checksums.txt 2>/dev/null || sha256sum *.tar.gz *.zip > checksums.txt
cd ..

print_message $GREEN "✓ Build complete! Version: ${VERSION}"
print_message $BLUE "Build artifacts in: ${BUILD_DIR}/"
ls -la ${BUILD_DIR}/