#!/bin/bash
# Create GitHub release with assets

set -e

# Configuration
REPO="elee1766/gofer"
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"
BUILD_DIR="dist"
CHANGELOG_FILE="CHANGELOG.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Print colored message
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Check prerequisites
check_prerequisites() {
    if ! command -v gh &> /dev/null; then
        print_message $RED "Error: GitHub CLI (gh) is not installed"
        print_message $YELLOW "Install with: brew install gh"
        exit 1
    fi
    
    if ! gh auth status &> /dev/null; then
        print_message $RED "Error: Not authenticated with GitHub"
        print_message $YELLOW "Run: gh auth login"
        exit 1
    fi
    
    if [ ! -d "$BUILD_DIR" ]; then
        print_message $RED "Error: Build directory not found"
        print_message $YELLOW "Run: ./scripts/release/build-release.sh"
        exit 1
    fi
}

# Generate release notes
generate_release_notes() {
    local version=$1
    local notes=""
    
    # Try to extract notes from CHANGELOG.md
    if [ -f "$CHANGELOG_FILE" ]; then
        # Extract section for this version
        notes=$(awk "/^## \[?${version}\]?/{flag=1; next} /^## /{flag=0} flag" "$CHANGELOG_FILE")
    fi
    
    # If no notes found, generate default
    if [ -z "$notes" ]; then
        notes="## What's Changed

Full changelog: https://github.com/${REPO}/commits/${version}

## Installation

See the [installation guide](https://github.com/${REPO}/blob/main/INSTALL.md) for detailed instructions.

### Quick Install

\`\`\`bash
# Using go install
go install github.com/${REPO}/cmd/gofer@${version}

# Using homebrew
brew tap elee1766/gofer
brew install gofer

# Download binary (Linux example)
curl -L https://github.com/${REPO}/releases/download/${version}/gofer-${version}-linux-amd64.tar.gz | tar xz
sudo mv gofer /usr/local/bin/
\`\`\`

## Checksums

Download \`checksums.txt\` to verify your download:

\`\`\`bash
sha256sum -c checksums.txt
\`\`\`"
    fi
    
    echo "$notes"
}

# Main execution
print_message $BLUE "Creating GitHub release for version ${VERSION}..."

# Check prerequisites
check_prerequisites

# Check if tag exists
if ! git rev-parse "$VERSION" >/dev/null 2>&1; then
    print_message $YELLOW "Tag ${VERSION} doesn't exist. Creating it..."
    git tag -a "$VERSION" -m "Release ${VERSION}"
    git push origin "$VERSION"
fi

# Generate release notes
print_message $BLUE "Generating release notes..."
RELEASE_NOTES=$(generate_release_notes "$VERSION")

# Create release draft
print_message $BLUE "Creating release draft..."
gh release create "$VERSION" \
    --repo "$REPO" \
    --title "Release ${VERSION}" \
    --notes "$RELEASE_NOTES" \
    --draft

# Upload assets
print_message $BLUE "Uploading release assets..."
for file in ${BUILD_DIR}/*; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        print_message $BLUE "Uploading ${filename}..."
        gh release upload "$VERSION" "$file" --repo "$REPO"
        print_message $GREEN "✓ Uploaded ${filename}"
    fi
done

print_message $GREEN "✓ Release draft created successfully!"
print_message $YELLOW "Visit https://github.com/${REPO}/releases to publish the release"