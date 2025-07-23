#!/bin/bash

# Test coverage script for gofer
set -e

COVERAGE_DIR="coverage"
COVERAGE_FILE="${COVERAGE_DIR}/coverage.out"
COVERAGE_HTML="${COVERAGE_DIR}/coverage.html"
COVERAGE_FUNC="${COVERAGE_DIR}/coverage-func.out"
COVERAGE_THRESHOLD=80

echo "🧪 Running test coverage analysis..."

# Create coverage directory
mkdir -p "${COVERAGE_DIR}"

# Run tests with coverage
echo "Running tests with coverage..."
go test -v -coverprofile="${COVERAGE_FILE}" -covermode=atomic ./...

# Generate coverage report
echo "Generating coverage reports..."
go tool cover -html="${COVERAGE_FILE}" -o "${COVERAGE_HTML}"
go tool cover -func="${COVERAGE_FILE}" > "${COVERAGE_FUNC}"

# Calculate total coverage
TOTAL_COVERAGE=$(go tool cover -func="${COVERAGE_FILE}" | tail -1 | awk '{print $3}' | sed 's/%//')

echo "📊 Coverage Report Generated:"
echo "  - HTML Report: ${COVERAGE_HTML}"
echo "  - Function Report: ${COVERAGE_FUNC}"
echo "  - Total Coverage: ${TOTAL_COVERAGE}%"

# Check coverage threshold
if (( $(echo "${TOTAL_COVERAGE} < ${COVERAGE_THRESHOLD}" | bc -l) )); then
    echo "❌ Coverage ${TOTAL_COVERAGE}% is below threshold ${COVERAGE_THRESHOLD}%"
    exit 1
else
    echo "✅ Coverage ${TOTAL_COVERAGE}% meets threshold ${COVERAGE_THRESHOLD}%"
fi

# Show top uncovered functions
echo ""
echo "🔍 Functions with lowest coverage:"
grep -v "total:" "${COVERAGE_FUNC}" | sort -k3 -n | head -10

echo ""
echo "📈 Coverage analysis complete!"
echo "Open ${COVERAGE_HTML} in your browser to view detailed coverage report."