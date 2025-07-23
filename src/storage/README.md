# Storage Package

The storage package provides a comprehensive SQLite-based storage layer for gofer, handling persistence for conversations, messages, settings, tool executions, sessions, and user preferences.

## Features

- **Thread-safe operations** with read-write mutex protection
- **Foreign key constraints** for data integrity
- **WAL mode** enabled for better concurrency
- **Comprehensive indexing** for optimal query performance
- **Builder patterns** for easy entity creation
- **Migration support** with schema versioning
- **JSON storage** for flexible data structures

## Usage

### Initialization

```go
// Create storage with file-based database
storage, err := storage.NewSQLiteStorage("./data/gofer.db")
if err != nil {
    log.Fatal(err)
}
defer storage.Close()

// Create storage with in-memory database (for testing)
storage, err := storage.NewSQLiteStorage(":memory:")
```

### Conversation Management

```go
// Create a conversation using the builder
conv := storage.NewConversationBuilder().
    WithTitle("My Conversation").
    WithModel("claude-3-opus-20240229").
    Build()

err := storage.CreateConversation(ctx, conv)

// List recent conversations
conversations, err := storage.ListConversations(ctx, 10, 0) // limit=10, offset=0
```

### Message Storage

```go
// Add a message to a conversation
msg := storage.NewMessageBuilder().
    WithConversationID(conv.ID).
    WithRole("user").
    WithContent("Hello, Claude!").
    Build()

err := storage.CreateMessage(ctx, msg)

// Retrieve conversation messages
messages, err := storage.GetMessagesByConversation(ctx, conv.ID, 100, 0)
```

### Settings Management

```go
// Set application settings
err := storage.SetSetting(ctx, "theme", "dark")

// Get a setting
setting, err := storage.GetSetting(ctx, "theme")
fmt.Println(setting.Value) // "dark"

// Apply default settings
for key, value := range storage.DefaultSettings() {
    storage.SetSetting(ctx, key, value)
}
```

### Tool Execution Logging

```go
// Log tool execution
exec := storage.NewToolExecutionBuilder().
    WithConversationID(conv.ID).
    WithToolName("file_reader").
    WithInput(map[string]interface{}{
        "path": "/path/to/file.txt",
    }).
    WithOutput(map[string]interface{}{
        "content": "file contents",
        "size": 1024,
    }).
    WithDuration(150). // milliseconds
    Build()

err := storage.LogToolExecution(ctx, exec)

// Query tool executions
hasError := false
filters := storage.ToolExecutionFilters{
    ToolName: &toolName,
    HasError: &hasError,
}
executions, err := storage.ListToolExecutions(ctx, filters, 10, 0)
```

### Session Management

```go
// Create a session
session := storage.NewSessionBuilder().
    WithUserID("user123").
    WithData(`{"authenticated": true}`).
    WithDuration(24 * time.Hour).
    Build()

err := storage.CreateSession(ctx, session)

// Clean up expired sessions
err := storage.DeleteExpiredSessions(ctx)
```

### User Preferences

```go
// Set user preferences
err := storage.SetUserPreference(ctx, userID, "editor_theme", "monokai")

// Get all preferences for a user
preferences, err := storage.GetUserPreferences(ctx, userID)

// Apply default user preferences
for key, value := range storage.DefaultUserPreferences() {
    storage.SetUserPreference(ctx, userID, key, value)
}
```

## Database Schema

The storage layer uses the following tables:

- `conversations` - Chat conversations with model information
- `messages` - Individual messages within conversations
- `settings` - Key-value configuration settings
- `tool_executions` - Logs of tool/function executions
- `sessions` - User session data with expiration
- `user_preferences` - User-specific preference settings
- `schema_migrations` - Database migration tracking

## Testing

Run the comprehensive test suite:

```bash
go test -v ./src/storage/...
```

## Performance Considerations

- WAL mode is enabled for better concurrent read performance
- All queries use prepared statements to prevent SQL injection
- Proper indexing on frequently queried columns
- Foreign key constraints ensure referential integrity
- Mutex protection prevents race conditions

## Error Handling

All methods return detailed errors with context:

```go
conv, err := storage.GetConversation(ctx, id)
if err != nil {
    // Error messages include operation context
    // e.g., "failed to get conversation: conversation not found"
}
```