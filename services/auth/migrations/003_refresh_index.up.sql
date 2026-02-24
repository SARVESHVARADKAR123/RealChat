CREATE INDEX idx_refresh_token_hash
ON refresh_tokens(token_hash);

CREATE INDEX idx_refresh_user_id
ON refresh_tokens(user_id);
