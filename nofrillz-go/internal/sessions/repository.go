package sessions

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, sessionID uint64, userID uint64) error {
	_, err := r.db.Exec(insertSessionQuery, sessionID, userID)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) GetActiveByID(ctx context.Context, sessionID uint64) (*Session, error) {
	session := Session{}

	err := r.db.QueryRow(selectActiveSessionByIDQuery, sessionID).Scan(&session.ID, &session.UserID, &session.Created, &session.Updated, &session.Revoked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error in database.QueryRow: %w", err)
	}

	return &session, nil
}

func (r *Repository) RevokeByID(ctx context.Context, sessionID uint64) (bool, error) {
	result, err := r.db.Exec(revokeSessionByIDQuery, sessionID)
	if err != nil {
		return false, fmt.Errorf("error in database.Exec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) CreateRefresh(ctx context.Context, tokenHash string, sessionID uint64, expiresAt time.Time) error {
	_, err := r.db.Exec(insertRefreshSessionQuery, tokenHash, sessionID, expiresAt.UTC())
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) GetActiveRefreshByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	refreshToken := RefreshToken{}

	err := r.db.QueryRow(selectActiveRefreshSessionByTokenQuery, tokenHash).Scan(
		&refreshToken.TokenHash,
		&refreshToken.SessionID,
		&refreshToken.UserID,
		&refreshToken.Created,
		&refreshToken.ExpiresAt,
		&refreshToken.Revoked,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error in database.QueryRow: %w", err)
	}

	return &refreshToken, nil
}

func (r *Repository) RevokeRefreshByTokenHash(ctx context.Context, tokenHash string) (bool, error) {
	result, err := r.db.Exec(revokeRefreshSessionByTokenQuery, tokenHash)
	if err != nil {
		return false, fmt.Errorf("error in database.Exec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) RevokeRefreshBySessionID(ctx context.Context, sessionID uint64) (bool, error) {
	result, err := r.db.Exec(revokeRefreshSessionsBySessionIDQuery, sessionID)
	if err != nil {
		return false, fmt.Errorf("error in database.Exec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}
