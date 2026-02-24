CREATE TABLE IF NOT EXISTS contacts (
    user_id UUID NOT NULL,
    contact_user_id UUID NOT NULL,
    nickname TEXT,
    is_favorite BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_contacts PRIMARY KEY (user_id, contact_user_id),
    CONSTRAINT chk_not_self_contact CHECK (user_id <> contact_user_id)
);

CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_contact_user_id ON contacts(contact_user_id);

CREATE INDEX IF NOT EXISTS idx_contacts_favorite
ON contacts(user_id, is_favorite)
WHERE is_favorite = TRUE;
