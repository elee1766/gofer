-- +goose Up
-- +goose StatementBegin

-- Conversations table
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    project_directory TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversations_project_dir ON conversations(project_directory);
CREATE INDEX idx_conversations_created_at ON conversations(created_at DESC);

-- Messages table
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,
    provider TEXT,
    model TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Tool executions table
CREATE TABLE tool_executions (
    id TEXT PRIMARY KEY,
    message_id TEXT,
    conversation_id TEXT,
    provider TEXT,
    model TEXT,
    tool_name TEXT NOT NULL,
    input TEXT NOT NULL, -- JSON
    output TEXT, -- JSON
    error TEXT,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE INDEX idx_tool_exec_conversation_id ON tool_executions(conversation_id);
CREATE INDEX idx_tool_exec_message_id ON tool_executions(message_id);
CREATE INDEX idx_tool_exec_created_at ON tool_executions(created_at DESC);

-- Settings table
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    current_conversation_id TEXT, -- current active conversation
    conversation_ids TEXT NOT NULL, -- comma-separated conversation ids (most recent last)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (current_conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_tool_exec_created_at;
DROP INDEX IF EXISTS idx_tool_exec_message_id;
DROP INDEX IF EXISTS idx_tool_exec_conversation_id;
DROP TABLE IF EXISTS tool_executions;

DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_conversation_id;
DROP TABLE IF EXISTS messages;

DROP INDEX IF EXISTS idx_conversations_created_at;
DROP INDEX IF EXISTS idx_conversations_project_dir;
DROP TABLE IF EXISTS conversations;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS settings;

-- +goose StatementEnd
