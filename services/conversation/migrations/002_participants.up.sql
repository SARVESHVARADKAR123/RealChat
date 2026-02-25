CREATE TABLE conversation_participants (
    conversation_id TEXT NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    role            participant_role NOT NULL DEFAULT 'member',
    last_read_sequence BIGINT NOT NULL DEFAULT 0,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_participants_user_id ON conversation_participants (user_id);
