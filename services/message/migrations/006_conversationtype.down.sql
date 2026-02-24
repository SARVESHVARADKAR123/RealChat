-- Reverse: remove the conversation_type column and enum added in 006.
-- Note: is_group was never part of the schema (001 never had it), so we do NOT restore it.

-- 1. Drop the enum column
ALTER TABLE conversations
    DROP COLUMN type;

-- 2. Drop the enum type
DROP TYPE conversation_type;
