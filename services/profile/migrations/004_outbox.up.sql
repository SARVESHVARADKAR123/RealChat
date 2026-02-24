CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY,
    topic TEXT NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP
);

-- Needed for efficient polling by publisher
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
ON outbox (created_at)
WHERE published_at IS NULL;
