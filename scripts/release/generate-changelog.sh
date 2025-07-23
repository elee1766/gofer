#!/bin/bash
# Generate changelog from git commits

set -e

# Configuration
CHANGELOG_FILE="CHANGELOG.md"
REPO_URL="https://github.com/elee1766/gofer"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored message
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Get all tags sorted by version
get_tags() {
    git tag -l --sort=-v:refname
}

# Get commits between two tags
get_commits() {
    local from=$1
    local to=$2
    
    if [ -z "$to" ]; then
        git log --pretty=format:"- %s (%h)" "$from"..HEAD
    else
        git log --pretty=format:"- %s (%h)" "$to".."$from"
    fi
}

# Categorize commits
categorize_commits() {
    local commits=$1
    local features=""
    local fixes=""
    local docs=""
    local tests=""
    local refactors=""
    local others=""
    
    while IFS= read -r commit; do
        if [[ $commit =~ ^-\ feat(\(.*\))?:\ .* ]]; then
            features="${features}${commit}\n"
        elif [[ $commit =~ ^-\ fix(\(.*\))?:\ .* ]]; then
            fixes="${fixes}${commit}\n"
        elif [[ $commit =~ ^-\ docs(\(.*\))?:\ .* ]]; then
            docs="${docs}${commit}\n"
        elif [[ $commit =~ ^-\ test(\(.*\))?:\ .* ]]; then
            tests="${tests}${commit}\n"
        elif [[ $commit =~ ^-\ refactor(\(.*\))?:\ .* ]]; then
            refactors="${refactors}${commit}\n"
        else
            others="${others}${commit}\n"
        fi
    done <<< "$commits"
    
    # Build categorized output
    local output=""
    
    if [ -n "$features" ]; then
        output="${output}### Features\n${features}\n"
    fi
    
    if [ -n "$fixes" ]; then
        output="${output}### Bug Fixes\n${fixes}\n"
    fi
    
    if [ -n "$refactors" ]; then
        output="${output}### Refactoring\n${refactors}\n"
    fi
    
    if [ -n "$docs" ]; then
        output="${output}### Documentation\n${docs}\n"
    fi
    
    if [ -n "$tests" ]; then
        output="${output}### Tests\n${tests}\n"
    fi
    
    if [ -n "$others" ]; then
        output="${output}### Other Changes\n${others}\n"
    fi
    
    echo -e "$output"
}

# Generate changelog entry for a version
generate_version_entry() {
    local version=$1
    local date=$2
    local prev_version=$3
    local commits=$(get_commits "$version" "$prev_version")
    
    if [ -z "$commits" ]; then
        return
    fi
    
    echo "## [$version] - $date"
    echo ""
    
    # Categorize and display commits
    categorize_commits "$commits"
    
    echo ""
}

# Main execution
print_message $BLUE "Generating changelog..."

# Create new changelog
cat > "$CHANGELOG_FILE" << EOF
# Changelog

All notable changes to gofer will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF

# Add unreleased section
unreleased_commits=$(get_commits "HEAD" "$(git describe --tags --abbrev=0 2>/dev/null || echo "")")
if [ -n "$unreleased_commits" ]; then
    echo "## [Unreleased]" >> "$CHANGELOG_FILE"
    echo "" >> "$CHANGELOG_FILE"
    categorize_commits "$unreleased_commits" >> "$CHANGELOG_FILE"
    echo "" >> "$CHANGELOG_FILE"
fi

# Process all tags
tags=($(get_tags))
for i in "${!tags[@]}"; do
    tag="${tags[$i]}"
    date=$(git log -1 --format=%cd --date=short "$tag")
    
    # Get previous tag
    prev_tag=""
    if [ $((i + 1)) -lt ${#tags[@]} ]; then
        prev_tag="${tags[$((i + 1))]}"
    fi
    
    generate_version_entry "$tag" "$date" "$prev_tag" >> "$CHANGELOG_FILE"
done

# Add links section
echo "[Unreleased]: ${REPO_URL}/compare/$(git describe --tags --abbrev=0)...HEAD" >> "$CHANGELOG_FILE"

for i in "${!tags[@]}"; do
    tag="${tags[$i]}"
    if [ $((i + 1)) -lt ${#tags[@]} ]; then
        prev_tag="${tags[$((i + 1))]}"
        echo "[$tag]: ${REPO_URL}/compare/${prev_tag}...${tag}" >> "$CHANGELOG_FILE"
    else
        echo "[$tag]: ${REPO_URL}/releases/tag/${tag}" >> "$CHANGELOG_FILE"
    fi
done

print_message $GREEN "✓ Changelog generated successfully!"
print_message $BLUE "Saved to: $CHANGELOG_FILE"