CREATE TABLE IF NOT EXISTS blocks (
    user_id UUID NOT NULL,
    blocked_user_id UUID NOT NULL,
    reason TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_blocks PRIMARY KEY (user_id, blocked_user_id),
    CONSTRAINT chk_not_self_block CHECK (user_id <> blocked_user_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_user_id ON blocks(user_id);
CREATE INDEX IF NOT EXISTS idx_blocks_blocked_user_id ON blocks(blocked_user_id);
