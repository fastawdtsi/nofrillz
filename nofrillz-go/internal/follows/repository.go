package follows

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

func (r *Repository) Follow(ctx context.Context, followerID uint64, followingID uint64) (bool, error) {
	result, err := r.db.ExecContext(ctx, insertFollowQuery, followerID, followingID)
	if err != nil {
		return false, fmt.Errorf("error in database.Exec: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) Unfollow(ctx context.Context, followerID uint64, followingID uint64) error {
	_, err := r.db.Exec(deleteFollowQuery, followerID, followingID)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) ListFollowers(ctx context.Context, userID uint64) ([]uint64, error) {
	rows, err := r.db.Query(selectFollowersQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	followers := []uint64{}
	for rows.Next() {
		var followerID uint64
		err = rows.Scan(&followerID)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		followers = append(followers, followerID)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return followers, nil
}

func (r *Repository) ListFollowersPage(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Follower, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if cursor == nil {
		rows, err = r.db.Query(selectFollowersPageQuery, userID, limit)
	} else {
		rows, err = r.db.Query(
			selectFollowersPageByCursorQuery,
			userID,
			*cursor,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	followers := []*Follower{}
	for rows.Next() {
		follower := Follower{}
		err = rows.Scan(&follower.FollowerID)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		followers = append(followers, &follower)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return followers, nil
}

func (r *Repository) ListFollowing(ctx context.Context, userID uint64) ([]uint64, error) {
	rows, err := r.db.Query(selectFollowingQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	following := []uint64{}
	for rows.Next() {
		var followingID uint64
		err = rows.Scan(&followingID)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		following = append(following, followingID)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return following, nil
}
