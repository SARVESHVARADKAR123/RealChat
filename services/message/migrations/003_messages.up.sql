CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL
        REFERENCES conversations(id) ON DELETE CASCADE,

    sender_id       TEXT NOT NULL,
    sequence        BIGINT NOT NULL,

    type            TEXT NOT NULL DEFAULT 'text',
    content         TEXT NOT NULL,
    metadata        JSONB,
    deleted_at      TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (conversation_id, sequence)
);

CREATE INDEX idx_messages_conv_seq_desc
ON messages(conversation_id, sequence DESC);

CREATE INDEX idx_messages_conv_sent_at_desc
ON messages(conversation_id, sent_at DESC);
