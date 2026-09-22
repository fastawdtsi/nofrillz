package apns

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByUserID(ctx context.Context, userID uint64) ([]*DeviceToken, error) {
	rows, err := r.db.QueryContext(ctx, selectTokensByUserIDQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	results := []*DeviceToken{}
	for rows.Next() {
		var token DeviceToken
		if err := rows.Scan(&token.Token, &token.UserID, &token.CreatedAt, &token.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		results = append(results, &token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}

func (r *Repository) Upsert(ctx context.Context, userID uint64, token string) error {
	if _, err := r.db.ExecContext(ctx, upsertTokenQuery, token, userID); err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) DeleteTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens))
	for _, token := range tokens {
		placeholders = append(placeholders, "?")
		args = append(args, token)
	}

	query := "delete from users_apns_device_tokens where token in (" + strings.Join(placeholders, ",") + ")"
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}
