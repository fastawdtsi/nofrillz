package users

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

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.CreateWithExecutor(ctx, r.db, user)
}

func (r *Repository) CreateWithExecutor(ctx context.Context, executor CreateExecutor, user *User) error {
	query := "insert into users (id,email,username,first_name,last_name,about,account_type,password_hash,password_salt,ai_model_preference) values (?,?,?,?,?,?,?,?,?,?)"
	_, err := executor.ExecContext(ctx, query, user.ID, user.Email, user.Username, user.FirstName, user.LastName, user.About, user.AccountType, user.PasswordHash, user.PasswordSalt, user.AIModelPreference)
	if err != nil {
		return fmt.Errorf("error in database.Exec: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, ID uint64) (*User, error) {
	user := User{}

	err := r.db.QueryRow(selectUserByIdQuery, ID).Scan(&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName, &user.About, &user.AccountType, &user.PasswordHash, &user.PasswordSalt, &user.Created, &user.Updated, &user.Deleted, &user.Blocked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error in database.QueryRow: %w", err)
		}
	}

	return &user, nil
}

func (r *Repository) GetByIDForRequester(ctx context.Context, ID uint64, requesterUserID uint64) (*User, error) {
	user := User{}

	err := r.db.QueryRow(selectUserByIdForRequesterQuery, requesterUserID, requesterUserID, ID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.About,
		&user.AccountType,
		&user.PasswordHash,
		&user.PasswordSalt,
		&user.Created,
		&user.Updated,
		&user.Deleted,
		&user.Blocked,
		&user.IsFollower,
		&user.IsFollowing,
		&user.FollowerCount,
		&user.FollowingCount,
		&user.PostCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error in database.QueryRow: %w", err)
		}
	}

	return &user, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := User{}

	err := r.db.QueryRow(selectUserByEmailQuery, email).Scan(&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName, &user.About, &user.AccountType, &user.PasswordHash, &user.PasswordSalt, &user.Created, &user.Updated, &user.Deleted, &user.Blocked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("error in database.QueryRow: %w", err)
		}
	}

	return &user, nil
}

func (r *Repository) SearchByUsernamePrefix(ctx context.Context, prefix string, limit int) ([]*User, error) {
	searchPrefix := prefix + "%"
	rows, err := r.db.Query(selectUsersByUsernamePrefixQuery, searchPrefix, searchPrefix, searchPrefix, limit)
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	results := []*User{}
	for rows.Next() {
		user := User{}
		err = rows.Scan(&user.ID, &user.Email, &user.Username, &user.FirstName, &user.LastName, &user.About, &user.AccountType, &user.PasswordHash, &user.PasswordSalt, &user.Created, &user.Updated, &user.Deleted, &user.Blocked)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		results = append(results, &user)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}

func (r *Repository) SearchByUsernamePrefixForRequester(ctx context.Context, prefix string, requesterUserID uint64, limit int) ([]*User, error) {
	searchPrefix := prefix + "%"
	rows, err := r.db.Query(selectUsersByUsernamePrefixForRequesterQuery, requesterUserID, requesterUserID, searchPrefix, searchPrefix, searchPrefix, limit)
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	results := []*User{}
	for rows.Next() {
		user := User{}
		err = rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.About,
			&user.AccountType,
			&user.PasswordHash,
			&user.PasswordSalt,
			&user.Created,
			&user.Updated,
			&user.Deleted,
			&user.Blocked,
			&user.IsFollower,
			&user.IsFollowing,
			&user.FollowerCount,
			&user.FollowingCount,
			&user.PostCount,
		)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		results = append(results, &user)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}

func (r *Repository) ListFollowersPageForRequester(ctx context.Context, userID uint64, requesterUserID uint64, cursor *uint64, limit int) ([]*User, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if cursor == nil {
		rows, err = r.db.Query(selectFollowersByUserForRequesterPageQuery, requesterUserID, requesterUserID, userID, limit)
	} else {
		rows, err = r.db.Query(selectFollowersByUserForRequesterPageByCursorQuery, requesterUserID, requesterUserID, userID, *cursor, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("error in database.Query: %w", err)
	}
	defer rows.Close()

	results := []*User{}
	for rows.Next() {
		user := User{}
		err = rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.About,
			&user.AccountType,
			&user.PasswordHash,
			&user.PasswordSalt,
			&user.Created,
			&user.Updated,
			&user.Deleted,
			&user.Blocked,
			&user.IsFollower,
			&user.IsFollowing,
			&user.FollowerCount,
			&user.FollowingCount,
			&user.PostCount,
		)
		if err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		results = append(results, &user)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}
