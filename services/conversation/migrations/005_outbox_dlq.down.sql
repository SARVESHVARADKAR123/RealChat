DROP TABLE IF EXISTS outbox_dlq;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS retry_count;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS error;
