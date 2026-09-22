package posts

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

func (r *Repository) Create(ctx context.Context, post *Post) error {
	return r.CreateWithExecutor(ctx, r.db, post)
}

func (r *Repository) CreateWithExecutor(ctx context.Context, executor CreateExecutor, post *Post) error {
	query := "insert into posts (id,user_id,body,source) values (?,?,?,?)"
	_, err := executor.ExecContext(ctx, query, post.ID, post.User.UserID, post.Body, post.Source)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, ID uint64, requesterUserID uint64) (*Post, error) {
	post := Post{}
	var username string
	var firstName string
	var lastName string
	var accountType string

	err := r.db.QueryRow(selectPostByIdQuery, requesterUserID, requesterUserID, ID).Scan(&post.ID, &post.User.UserID, &post.Body, &post.Source, &post.Created, &post.Updated, &post.Deleted, &username, &firstName, &lastName, &accountType, &post.Liked, &post.IsBookmarked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error in database.QueryRow: %w", err)
		}
	}

	post.User = PostUser{
		UserID:      post.User.UserID,
		Username:    username,
		FirstName:   firstName,
		LastName:    lastName,
		AccountType: accountType,
	}

	return &post, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID uint64, requesterUserID uint64, beforeID uint64, limit int) ([]*Post, error) {
	var rows *sql.Rows
	var err error

	if beforeID > 0 {
		rows, err = r.db.Query(selectPostsByUserIdBeforeIdQuery, requesterUserID, requesterUserID, userID, beforeID, limit)
	} else {
		rows, err = r.db.Query(selectPostsByUserIdQuery, requesterUserID, requesterUserID, userID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	posts := []*Post{}

	for rows.Next() {
		post := Post{}
		var username string
		var firstName string
		var lastName string
		var accountType string
		err = rows.Scan(&post.ID, &post.User.UserID, &post.Body, &post.Source, &post.Created, &post.Updated, &post.Deleted, &username, &firstName, &lastName, &accountType, &post.Liked, &post.IsBookmarked)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		post.User = PostUser{
			UserID:      post.User.UserID,
			Username:    username,
			FirstName:   firstName,
			LastName:    lastName,
			AccountType: accountType,
		}

		posts = append(posts, &post)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return posts, nil
}
