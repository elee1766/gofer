#!/bin/bash
# Main release script - orchestrates the entire release process

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

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

# Display usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] VERSION

Create a new release of gofer

Arguments:
  VERSION    Version to release (e.g., v1.2.3)

Options:
  -h, --help           Show this help message
  -s, --skip-tests     Skip running tests
  -b, --skip-build     Skip building binaries
  -c, --skip-changelog Skip changelog generation
  -p, --publish        Publish release immediately (not draft)
  -d, --dry-run        Run without making changes

Examples:
  $0 v1.2.3            Create release v1.2.3
  $0 -s v1.2.3         Skip tests and create release
  $0 -p v1.2.3         Create and publish release

EOF
}

# Parse command line arguments
SKIP_TESTS=false
SKIP_BUILD=false
SKIP_CHANGELOG=false
PUBLISH=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -s|--skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        -b|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        -c|--skip-changelog)
            SKIP_CHANGELOG=true
            shift
            ;;
        -p|--publish)
            PUBLISH=true
            shift
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -*)
            print_message $RED "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            VERSION=$1
            shift
            ;;
    esac
done

# Validate version
if [ -z "$VERSION" ]; then
    print_message $RED "Error: VERSION is required"
    usage
    exit 1
fi

# Validate version format
if ! [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    print_message $RED "Error: Invalid version format. Expected: vX.Y.Z or vX.Y.Z-suffix"
    exit 1
fi

# Change to project root
cd "$PROJECT_ROOT"

# Export version for other scripts
export VERSION

print_message $BLUE "Starting release process for ${VERSION}..."

# Step 1: Check prerequisites
print_message $BLUE "Checking prerequisites..."

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_message $RED "Error: Uncommitted changes found"
    print_message $YELLOW "Commit or stash your changes before releasing"
    exit 1
fi

# Check if on main branch
current_branch=$(git rev-parse --abbrev-ref HEAD)
if [ "$current_branch" != "main" ] && [ "$current_branch" != "master" ]; then
    print_message $YELLOW "Warning: Not on main/master branch (current: $current_branch)"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    print_message $RED "Error: Tag $VERSION already exists"
    exit 1
fi

print_message $GREEN "✓ Prerequisites check passed"

# Step 2: Run tests
if [ "$SKIP_TESTS" = false ]; then
    print_message $BLUE "Running tests..."
    if [ "$DRY_RUN" = false ]; then
        make test
        print_message $GREEN "✓ All tests passed"
    else
        print_message $YELLOW "→ Would run: make test"
    fi
else
    print_message $YELLOW "→ Skipping tests"
fi

# Step 3: Generate changelog
if [ "$SKIP_CHANGELOG" = false ]; then
    print_message $BLUE "Generating changelog..."
    if [ "$DRY_RUN" = false ]; then
        "${SCRIPT_DIR}/generate-changelog.sh"
        
        # Commit changelog if changed
        if ! git diff --quiet CHANGELOG.md; then
            git add CHANGELOG.md
            git commit -m "docs: update changelog for ${VERSION}"
            print_message $GREEN "✓ Changelog updated and committed"
        fi
    else
        print_message $YELLOW "→ Would run: generate-changelog.sh"
    fi
else
    print_message $YELLOW "→ Skipping changelog generation"
fi

# Step 4: Create and push tag
print_message $BLUE "Creating tag ${VERSION}..."
if [ "$DRY_RUN" = false ]; then
    git tag -a "$VERSION" -m "Release ${VERSION}"
    git push origin "$VERSION"
    print_message $GREEN "✓ Tag created and pushed"
else
    print_message $YELLOW "→ Would create tag: $VERSION"
fi

# Step 5: Build release binaries
if [ "$SKIP_BUILD" = false ]; then
    print_message $BLUE "Building release binaries..."
    if [ "$DRY_RUN" = false ]; then
        "${SCRIPT_DIR}/build-release.sh"
        print_message $GREEN "✓ Binaries built successfully"
    else
        print_message $YELLOW "→ Would run: build-release.sh"
    fi
else
    print_message $YELLOW "→ Skipping binary build"
fi

# Step 6: Create GitHub release
print_message $BLUE "Creating GitHub release..."
if [ "$DRY_RUN" = false ]; then
    if [ "$PUBLISH" = true ]; then
        # Modify the create-github-release.sh call to publish immediately
        sed -i.bak 's/--draft//' "${SCRIPT_DIR}/create-github-release.sh"
        "${SCRIPT_DIR}/create-github-release.sh"
        mv "${SCRIPT_DIR}/create-github-release.sh.bak" "${SCRIPT_DIR}/create-github-release.sh"
        print_message $GREEN "✓ Release published successfully!"
    else
        "${SCRIPT_DIR}/create-github-release.sh"
        print_message $GREEN "✓ Release draft created successfully!"
    fi
else
    print_message $YELLOW "→ Would create GitHub release"
fi

# Summary
print_message $GREEN "\n✓ Release process completed successfully!"
print_message $BLUE "Version: ${VERSION}"

if [ "$DRY_RUN" = true ]; then
    print_message $YELLOW "\nThis was a dry run. No changes were made."
elif [ "$PUBLISH" = false ]; then
    print_message $YELLOW "\nRelease created as draft. Visit GitHub to publish it."
fi

print_message $BLUE "\nNext steps:"
print_message $BLUE "1. Update package managers (Homebrew, AUR, etc.)"
print_message $BLUE "2. Announce the release"
print_message $BLUE "3. Update documentation if needed"