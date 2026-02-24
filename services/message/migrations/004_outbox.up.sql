CREATE TABLE outbox_events (
    id              BIGSERIAL PRIMARY KEY,

    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,

    payload         BYTEA NOT NULL,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unprocessed
ON outbox_events(created_at)
WHERE processed_at IS NULL;

