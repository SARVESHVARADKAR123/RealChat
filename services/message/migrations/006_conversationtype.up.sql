-- Add the conversation_type enum and typed column.
-- Note: Migration 001 already excludes is_group, so no data migration needed.

-- 1. Create enum type
CREATE TYPE conversation_type AS ENUM ('direct', 'group');

-- 2. Add the typed column (NOT NULL, required on all new rows)
ALTER TABLE conversations
    ADD COLUMN type conversation_type NOT NULL DEFAULT 'direct';

-- 3. Drop the default now that the column is established
ALTER TABLE conversations
    ALTER COLUMN type DROP DEFAULT;
