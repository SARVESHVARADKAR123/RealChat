-- Add lookup_key column to conversations table for semantic uniqueness of direct chats.

ALTER TABLE conversations
    ADD COLUMN lookup_key TEXT;

-- For DIRECT conversations, lookup_key = 'direct:user_a:user_b' (sorted).
-- For GROUP conversations, lookup_key remains NULL.
-- PostgreSQL UNIQUE constraint treats multiple NULLs as unique (allows them).

CREATE UNIQUE INDEX idx_conversations_lookup_key ON conversations(lookup_key) WHERE lookup_key IS NOT NULL;
