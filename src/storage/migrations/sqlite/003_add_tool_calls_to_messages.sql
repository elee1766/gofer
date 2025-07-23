-- +goose Up
-- +goose StatementBegin

-- Add tool_calls column to messages table
ALTER TABLE messages ADD COLUMN tool_calls TEXT; -- JSON array of tool calls

-- Create an index for messages that have tool calls
CREATE INDEX idx_messages_has_tool_calls ON messages(conversation_id) WHERE tool_calls IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the index
DROP INDEX IF EXISTS idx_messages_has_tool_calls;

-- SQLite doesn't support dropping columns directly, so we need to recreate the table
CREATE TABLE messages_new (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,
    provider TEXT,
    model TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Copy data from old table (excluding tool_calls)
INSERT INTO messages_new (id, conversation_id, role, provider, model, content, created_at)
SELECT id, conversation_id, role, provider, model, content, created_at
FROM messages;

-- Drop old table and rename new one
DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;

-- Recreate indexes
CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- +goose StatementEnd