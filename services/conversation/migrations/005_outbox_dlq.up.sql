ALTER TABLE outbox_events ADD COLUMN error TEXT;
ALTER TABLE outbox_events ADD COLUMN retry_count INT NOT NULL DEFAULT 0;

CREATE TABLE outbox_dlq (
    id             BIGINT PRIMARY KEY,
    aggregate_type TEXT        NOT NULL,
    aggregate_id   TEXT        NOT NULL,
    event_type     TEXT        NOT NULL,
    payload        BYTEA       NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    failed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    error          TEXT,
    retry_count    INT         NOT NULL DEFAULT 0
);
