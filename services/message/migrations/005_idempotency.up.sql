CREATE TABLE idempotency_keys (
    key              TEXT NOT NULL,
    user_id          TEXT NOT NULL,

    payload          JSONB,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (key, user_id)
);

CREATE INDEX idx_idempotency_user
ON idempotency_keys(user_id);
