-- Migration 012: Drop stale conversation schema from messaging database
--
-- Context: When the conversation service was split out into its own
-- microservice, the tables below (conversations, conversation_participants,
-- conversation_sequences) became the sole responsibility of the conversation
-- service and its own database.  They were left behind in the messaging DB.

-- Drop the stale tables in reverse dependency order.
-- Use CASCADE to ensure any hidden foreign keys or dependent objects are also removed.
DROP TABLE IF EXISTS conversation_sequences CASCADE;
DROP TABLE IF EXISTS conversation_participants CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;

-- Drop the stale enum type that was added for the conversations table.
DROP TYPE IF EXISTS conversation_type CASCADE;
