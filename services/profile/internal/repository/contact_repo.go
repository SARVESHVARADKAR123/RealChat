package repository

import (
	"context"
	"database/sql"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
)

// ContactRepo handles CRUD operations on the contacts table.
type ContactRepo struct{ DB *sql.DB }

// Add inserts a contact relationship (idempotent via ON CONFLICT DO NOTHING).
func (r *ContactRepo) Add(ctx context.Context, userID, contactUserID string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO contacts (user_id, contact_user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, contactUserID)
	return err
}

// Remove deletes a contact relationship.
func (r *ContactRepo) Remove(ctx context.Context, userID, contactUserID string) error {
	_, err := r.DB.ExecContext(ctx,
		`DELETE FROM contacts WHERE user_id = $1 AND contact_user_id = $2`,
		userID, contactUserID)
	return err
}

// List returns a paginated list of contacts for a user.
func (r *ContactRepo) List(ctx context.Context, userID string, limit, offset int) ([]model.Contact, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT contact_user_id, created_at
		FROM contacts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []model.Contact
	for rows.Next() {
		var c model.Contact
		c.UserID = userID
		if err := rows.Scan(&c.ContactID, &c.CreatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}
