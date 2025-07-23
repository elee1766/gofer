-- +goose Up
-- +goose StatementBegin

-- Create a new sessions table with JSON array for conversation_ids
CREATE TABLE sessions_new (
    id TEXT PRIMARY KEY,
    current_conversation_id TEXT,
    conversation_ids TEXT NOT NULL DEFAULT '[]', -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (current_conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

-- Copy existing data, converting comma-separated to JSON array
INSERT INTO sessions_new (id, current_conversation_id, conversation_ids, created_at, updated_at)
SELECT 
    id,
    current_conversation_id,
    CASE 
        WHEN conversation_ids = '' THEN '[]'
        WHEN conversation_ids NOT LIKE '%,%' THEN json_array(conversation_ids)
        ELSE '["' || replace(conversation_ids, ',', '","') || '"]'
    END,
    created_at,
    updated_at
FROM sessions;

-- Drop old table and rename new one
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Create old table structure
CREATE TABLE sessions_old (
    id TEXT PRIMARY KEY,
    current_conversation_id TEXT,
    conversation_ids TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (current_conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

-- Copy data back, converting JSON array to comma-separated
INSERT INTO sessions_old (id, current_conversation_id, conversation_ids, created_at, updated_at)
SELECT 
    id,
    current_conversation_id,
    COALESCE(
        (SELECT group_concat(value, ',') FROM json_each(conversation_ids)),
        ''
    ),
    created_at,
    updated_at
FROM sessions;

-- Drop new table and rename old one back
DROP TABLE sessions;
ALTER TABLE sessions_old RENAME TO sessions;

-- +goose StatementEnd