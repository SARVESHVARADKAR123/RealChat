CREATE TYPE conversation_type AS ENUM ('direct', 'group');
CREATE TYPE participant_role AS ENUM ('member', 'admin');

CREATE TABLE conversations (
    id           TEXT PRIMARY KEY,
    type         conversation_type NOT NULL,
    display_name TEXT,
    avatar_url   TEXT,
    lookup_key   TEXT UNIQUE,  -- used to deduplicate direct conversations
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_updated_at ON conversations (updated_at DESC);
