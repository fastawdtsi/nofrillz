package notifications

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByUserID(ctx context.Context, userID uint64) (*Settings, error) {
	var settings Settings
	if err := r.db.QueryRowContext(ctx, selectSettingsByUserIDQuery, userID).Scan(
		&settings.UserID,
		&settings.Enabled,
		&settings.NewFollowers,
		&settings.NewLikes,
		&settings.Replies,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error in database.QueryRow: %w", err)
	}

	return &settings, nil
}

func (r *Repository) Upsert(ctx context.Context, settings *Settings) error {
	if _, err := r.db.ExecContext(ctx, upsertSettingsQuery,
		settings.UserID,
		settings.Enabled,
		settings.NewFollowers,
		settings.NewLikes,
		settings.Replies,
	); err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}
