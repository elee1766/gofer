# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build and Run
- `make build` - Build the gofer binary
- `make install` - Build and install binary to /usr/local/bin/
- `./gofer` - Run the built binary

### Testing
- `make test` - Run all tests (unit + integration)
- `make test-unit` - Run unit tests only (`go test ./src/...`)
- `make test-integration` - Run integration tests only (`go test ./tests/integration/...`)
- `make test-coverage` - Generate coverage report (creates coverage.html)
- `make test-race` - Run tests with race detection

### Code Quality
- `make lint` - Run golangci-lint
- `make lint-fix` - Run linter with auto-fixes
- `make check` - Run lint, test, and build (comprehensive check)

### Development Setup
- `make deps` - Download and verify dependencies
- `make tidy` - Tidy go.mod
- `make install-hooks` - Install pre-commit hooks

## Architecture Overview

### Core Components

**App Layer** (`src/app/`) - Main application initialization and service coordination
- Manages ModelProvider (OpenRouter client), storage, and configuration
- Initializes SQLite database in `.gofer/sqlite.db`

**Agent System** (`src/agent/`, `src/goferagent/`) - AI agent implementation
- `src/agent/` - Generic agent framework with tool support
- `src/goferagent/` - Gofer-specific agent with comprehensive tool suite
- Tools include file operations, command execution, web fetching, and text processing

**Executor** (`src/executor/`) - Conversation orchestration and streaming
- Handles step-by-step conversation execution with tool calls
- Manages streaming responses and event processing
- Coordinates between model client and tool execution

**Storage** (`src/storage/`) - SQLite-based persistence layer
- Conversations, messages, sessions, tool executions, settings
- Uses WAL mode for better concurrency
- Migration system for schema versioning

**Configuration** (`src/config/`) - Hierarchical configuration management
- Loads from system, user, project, local, env vars, and CLI args
- Comprehensive permission system for tools, filesystem, and commands
- Security controls and audit logging

**AI SDK** (`src/aisdk/`) - OpenRouter integration layer
- Model client abstraction with streaming support
- Chat completion requests and tool calling
- Provider-agnostic interface

### Tool System

The tool system (`src/goferagent/tools/`) provides:
- File operations (read, write, edit, patch, copy, move, delete)
- Directory operations (list, create, info)
- Search and grep functionality
- Command execution with output capture
- Web fetching capabilities

Each tool has comprehensive tests and follows a consistent interface pattern.

### Key CLI Commands

- `gofer prompt "message"` - Execute single prompt
- `gofer migrate` - Run database migrations  
- `gofer model` - Model management and information

### Storage Locations

- Config: `.gofer/config.json` (hierarchical discovery)
- Database: `.gofer/sqlite.db` 
- Logs: Pre-commit hooks log to git hooks

### Pre-commit Hooks

The repository includes comprehensive pre-commit validation:
- Go formatting and imports check
- Linting with golangci-lint
- Build verification
- Test execution
- Optional security scanning with gosec