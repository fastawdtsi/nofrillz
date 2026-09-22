package bookmarks

import (
	"context"
	"database/sql"
	"fmt"

	"nofrillz/internal/posts"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	result, err := r.db.ExecContext(ctx, insertBookmarkQuery, userID, postID)
	if err != nil {
		return false, fmt.Errorf("error in database.ExecContext: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) UnbookmarkPost(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	result, err := r.db.ExecContext(ctx, deleteBookmarkQuery, userID, postID)
	if err != nil {
		return false, fmt.Errorf("error in database.ExecContext: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) IsPostBookmarked(ctx context.Context, postID uint64, userID uint64) (bool, error) {
	var bookmarked bool
	if err := r.db.QueryRowContext(ctx, selectBookmarkExistsQuery, userID, postID).Scan(&bookmarked); err != nil {
		return false, fmt.Errorf("error in database.QueryRowContext: %w", err)
	}

	return bookmarked, nil
}

func (r *Repository) ListBookmarkedPosts(ctx context.Context, userID uint64, cursor *Cursor, limit int) ([]*posts.Post, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if cursor == nil {
		rows, err = r.db.QueryContext(ctx, selectBookmarkedPostsQuery, userID, userID, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, selectBookmarkedPostsByCursorQuery, userID, userID, cursor.Created, cursor.Created, cursor.PostID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("error in database.QueryContext: %w", err)
	}
	defer rows.Close()

	results := make([]*posts.Post, 0, limit)
	for rows.Next() {
		post := posts.Post{}
		var bookmarkedAt sql.NullTime
		err = rows.Scan(
			&post.ID,
			&post.User.UserID,
			&post.Body,
			&post.Source,
			&post.Created,
			&post.Updated,
			&post.Deleted,
			&post.User.Username,
			&post.User.FirstName,
			&post.User.LastName,
			&post.User.AccountType,
			&post.Liked,
			&post.IsBookmarked,
			&bookmarkedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		if bookmarkedAt.Valid {
			value := bookmarkedAt.Time
			post.BookmarkedAt = &value
		}
		results = append(results, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}
