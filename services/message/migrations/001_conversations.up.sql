CREATE TABLE conversations (
    id          TEXT PRIMARY KEY,

    -- optional product metadata
    display_name  TEXT,
    avatar_url    TEXT,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);




CREATE INDEX idx_conversations_updated_at
ON conversations(updated_at DESC);
