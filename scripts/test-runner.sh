#!/bin/bash

# Test runner script for gofer
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
VERBOSE=false
COVERAGE=false
INTEGRATION=false
RACE=false
LINT=false
SECURITY=false
CLEAN=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -i|--integration)
            INTEGRATION=true
            shift
            ;;
        -r|--race)
            RACE=true
            shift
            ;;
        -l|--lint)
            LINT=true
            shift
            ;;
        -s|--security)
            SECURITY=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -v, --verbose     Enable verbose output"
            echo "  -c, --coverage    Run with coverage"
            echo "  -i, --integration Run integration tests"
            echo "  -r, --race        Run with race detection"
            echo "  -l, --lint        Run linter"
            echo "  -s, --security    Run security scanner"
            echo "  --clean           Clean test artifacts"
            echo "  -h, --help        Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option $1"
            exit 1
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Print header
echo -e "${BLUE}🧪 GoCodeCLI Test Runner${NC}"
echo "================================="

# Clean artifacts if requested
if [ "$CLEAN" = true ]; then
    echo -e "${YELLOW}🧹 Cleaning test artifacts...${NC}"
    rm -rf coverage/
    rm -f *.out *.html *.prof
    echo -e "${GREEN}✅ Clean complete${NC}"
    exit 0
fi

# Build test flags
TEST_FLAGS=""
if [ "$VERBOSE" = true ]; then
    TEST_FLAGS="$TEST_FLAGS -v"
fi
if [ "$RACE" = true ]; then
    TEST_FLAGS="$TEST_FLAGS -race"
fi

# Run linter
if [ "$LINT" = true ]; then
    echo -e "${YELLOW}🔍 Running linter...${NC}"
    if golangci-lint run; then
        echo -e "${GREEN}✅ Linter passed${NC}"
    else
        echo -e "${RED}❌ Linter failed${NC}"
        exit 1
    fi
    echo ""
fi

# Run security scanner
if [ "$SECURITY" = true ]; then
    echo -e "${YELLOW}🔒 Running security scanner...${NC}"
    if command -v gosec &> /dev/null; then
        if gosec ./...; then
            echo -e "${GREEN}✅ Security scan passed${NC}"
        else
            echo -e "${RED}❌ Security scan failed${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}⚠️  gosec not installed, skipping security scan${NC}"
    fi
    echo ""
fi

# Run unit tests
echo -e "${YELLOW}🧪 Running unit tests...${NC}"
if [ "$COVERAGE" = true ]; then
    mkdir -p coverage
    if go test $TEST_FLAGS -coverprofile=coverage/coverage.out ./src/...; then
        echo -e "${GREEN}✅ Unit tests passed${NC}"
        
        # Generate coverage report
        go tool cover -html=coverage/coverage.out -o coverage/coverage.html
        COVERAGE_PCT=$(go tool cover -func=coverage/coverage.out | tail -1 | awk '{print $3}')
        echo -e "${BLUE}📊 Coverage: $COVERAGE_PCT${NC}"
        echo -e "${BLUE}📄 HTML report: coverage/coverage.html${NC}"
    else
        echo -e "${RED}❌ Unit tests failed${NC}"
        exit 1
    fi
else
    if go test $TEST_FLAGS ./src/...; then
        echo -e "${GREEN}✅ Unit tests passed${NC}"
    else
        echo -e "${RED}❌ Unit tests failed${NC}"
        exit 1
    fi
fi
echo ""

# Run integration tests
if [ "$INTEGRATION" = true ]; then
    echo -e "${YELLOW}🔗 Running integration tests...${NC}"
    if go test $TEST_FLAGS -tags=integration ./tests/integration/...; then
        echo -e "${GREEN}✅ Integration tests passed${NC}"
    else
        echo -e "${RED}❌ Integration tests failed${NC}"
        exit 1
    fi
    echo ""
fi

# Build check
echo -e "${YELLOW}🏗️  Building project...${NC}"
if go build ./cmd/gofer; then
    echo -e "${GREEN}✅ Build successful${NC}"
    rm -f gofer  # Clean up binary
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 All tests completed successfully!${NC}"