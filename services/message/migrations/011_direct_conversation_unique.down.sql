-- Drop unique index and column.

DROP INDEX IF EXISTS idx_conversations_lookup_key;

ALTER TABLE conversations
    DROP COLUMN IF EXISTS lookup_key;
