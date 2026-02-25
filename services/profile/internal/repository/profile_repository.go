package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
)

type ProfileRepo struct{ DB *sql.DB }

func (r *ProfileRepo) CreateIfNotExists(ctx context.Context, id string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO profiles(user_id, username) VALUES($1::uuid, $2::text) ON CONFLICT DO NOTHING`, id, id)
	return err
}

func (r *ProfileRepo) Get(ctx context.Context, id string) (*model.Profile, error) {
	p := &model.Profile{}
	var displayName, avatarURL, bio sql.NullString
	err := r.DB.QueryRowContext(ctx,
		`SELECT user_id, username, display_name, avatar_url, bio, created_at, updated_at FROM profiles WHERE user_id=$1::uuid`, id).
		Scan(&p.UserID, &p.Username, &displayName, &avatarURL, &bio, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, model.ErrProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %w", err)
	}

	p.DisplayName = displayName.String
	p.AvatarURL = avatarURL.String
	p.Bio = bio.String

	return p, nil
}

func (r *ProfileRepo) BatchGet(ctx context.Context, ids []string) ([]*model.Profile, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Simple implementation using ANY($1) for better performance than multiple IN params
	rows, err := r.DB.QueryContext(ctx,
		`SELECT user_id, username, display_name, avatar_url, bio, created_at, updated_at 
		 FROM profiles 
		 WHERE user_id::text = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*model.Profile
	for rows.Next() {
		p := &model.Profile{}
		var displayName, avatarURL, bio sql.NullString
		if err := rows.Scan(&p.UserID, &p.Username, &displayName, &avatarURL, &bio, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		p.DisplayName = displayName.String
		p.AvatarURL = avatarURL.String
		p.Bio = bio.String
		profiles = append(profiles, p)
	}

	return profiles, nil
}

func (r *ProfileRepo) Update(ctx context.Context, p *model.Profile) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE profiles SET display_name=$2,bio=$3,avatar_url=$4,updated_at=NOW() WHERE user_id=$1`,
		p.UserID, p.DisplayName, p.Bio, p.AvatarURL)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrProfileNotFound
	}
	return nil
}
