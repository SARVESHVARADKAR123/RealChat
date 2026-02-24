CREATE TABLE conversation_sequences (
    conversation_id TEXT PRIMARY KEY
        REFERENCES conversations(id) ON DELETE CASCADE,

    next_sequence   BIGINT NOT NULL
);

CREATE TABLE conversation_participants (
    conversation_id TEXT NOT NULL
        REFERENCES conversations(id) ON DELETE CASCADE,

    user_id     TEXT NOT NULL,
    last_read_sequence BIGINT NOT NULL DEFAULT 0,
    role        TEXT NOT NULL DEFAULT 'member',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_participants_user_conv
ON conversation_participants(user_id, conversation_id);
