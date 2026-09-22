package likes

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Like(ctx context.Context, postID uint64, userID uint64) error {
	_, err := r.db.Exec(insertLikeQuery, postID, userID)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) Unlike(ctx context.Context, postID uint64, userID uint64) error {
	_, err := r.db.Exec(deleteLikeQuery, postID, userID)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}
