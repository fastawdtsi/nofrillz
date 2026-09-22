package feed

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

func (r *Repository) List(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if cursor == nil {
		rows, err = r.db.QueryContext(
			ctx,
			selectFeedPostsQuery,
			requesterUserID,
			requesterUserID,
			requesterUserID,
			requesterUserID,
			limit,
		)
	} else {
		rows, err = r.db.QueryContext(
			ctx,
			selectFeedPostsByCursorQuery,
			requesterUserID,
			requesterUserID,
			requesterUserID,
			requesterUserID,
			*cursor,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows)
}

func scanPosts(rows *sql.Rows) ([]*Post, error) {
	posts := []*Post{}
	for rows.Next() {
		post := Post{}
		err := rows.Scan(&post.ID, &post.Body, &post.Source, &post.Created, &post.Liked, &post.IsBookmarked, &post.UserID, &post.Username, &post.FirstName, &post.LastName, &post.AccountType)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		posts = append(posts, &post)
	}

	err := rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return posts, nil
}

func (r *Repository) ListDiscover(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error) {
	query := selectDiscoverPostsQuery
	args := []any{requesterUserID, requesterUserID}
	if cursor != nil {
		query += " AND p.id < ?"
		args = append(args, *cursor)
	}
	query += " ORDER BY p.id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}
