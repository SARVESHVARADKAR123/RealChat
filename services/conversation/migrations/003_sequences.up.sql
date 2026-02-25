CREATE TABLE conversation_sequences (
    conversation_id TEXT PRIMARY KEY REFERENCES conversations (id) ON DELETE CASCADE,
    next_sequence   BIGINT NOT NULL DEFAULT 0
);
