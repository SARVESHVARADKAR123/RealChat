-- Migration to scope idempotency_key to conversation_id

-- 1. Add the column
ALTER TABLE idempotency_keys
ADD COLUMN conversation_id TEXT;

-- 2. Populate conversation_id if any existing keys exist (unlikely to be useful without the actual ID, but we need it for PK)
-- Since this is a dev project, we can assume we can fill with a placeholder or just truncate.
-- Let's use a dummy value for existing rows so we can make it NOT NULL.
UPDATE idempotency_keys SET conversation_id = 'legacy' WHERE conversation_id IS NULL;

-- 3. Make it NOT NULL
ALTER TABLE idempotency_keys ALTER COLUMN conversation_id SET NOT NULL;

-- 4. Update the Primary Key
ALTER TABLE idempotency_keys DROP CONSTRAINT idempotency_keys_pkey;
ALTER TABLE idempotency_keys ADD PRIMARY KEY (key, user_id, conversation_id);

-- 5. Add index for performance
CREATE INDEX idx_idempotency_conversation ON idempotency_keys(conversation_id);
