package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"nofrillz/internal/users"
)

const selectStatsQuery = `
select
	(select count(*) from users where deleted is null and blocked is null) as user_count,
	(select count(*) from users where deleted is null and blocked is null and account_type='human') as human_user_count,
	(select count(*) from users where deleted is null and blocked is null and account_type='ai') as ai_user_count,
	(select count(*) from users where deleted is null and blocked is not null) as blocked_user_count,
	(
		select count(*)
		from posts p
		join users u on u.id = p.user_id
		where p.deleted is null
		  and u.deleted is null
		  and u.blocked is null
	) as post_count,
	(
		select count(*)
		from posts p
		join users u on u.id = p.user_id
		where p.deleted is null
		  and u.deleted is null
		  and u.blocked is null
		  and p.source = 'ai'
	) as ai_post_count,
	(
		select count(*)
		from ai_accounts a
		join users u on u.id = a.user_id
		where a.enabled = true
		  and u.deleted is null
		  and u.blocked is null
	) as enabled_ai_account_count
`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Stats(ctx context.Context) (*Stats, error) {
	stats := Stats{}
	if err := r.db.QueryRowContext(ctx, selectStatsQuery).Scan(
		&stats.UserCount,
		&stats.HumanUserCount,
		&stats.AIUserCount,
		&stats.BlockedUserCount,
		&stats.PostCount,
		&stats.AIPostCount,
		&stats.EnabledAIAccountCount,
	); err != nil {
		return nil, fmt.Errorf("error in database.QueryRowContext: %w", err)
	}

	return &stats, nil
}

func (r *Repository) ListUsers(ctx context.Context, input ListUsersInput) ([]*UserSummary, error) {
	args := make([]any, 0, 8)
	query := strings.Builder{}
	query.WriteString(`
select
	u.id,
	u.email,
	u.username,
	u.first_name,
	u.last_name,
	u.about,
	u.account_type,
	u.blocked,
	u.created,
	(
		select count(*)
		from posts p
		where p.user_id = u.id
		  and p.deleted is null
	) as post_count,
	a.id as ai_account_id,
 COALESCE(a.enabled, false), a.next_generate_at, a.last_generated_at,
 COALESCE(a.generation_status, ''), COALESCE(a.generation_error, ''),
 COALESCE(a.min_posts_per_day, 0), COALESCE(a.max_posts_per_day, 0), COALESCE(a.content_mode,''),COALESCE(a.check_interval_seconds,0),a.last_checked_at,COALESCE(a.last_check_outcome,''),COALESCE(a.model_options,JSON_ARRAY())
from users u
left join ai_accounts a on a.user_id = u.id
where u.deleted is null
`)

	if input.Query != "" {
		search := "%" + input.Query + "%"
		query.WriteString(`
  and (
	u.username like ?
	or u.email like ?
	or u.first_name like ?
	or u.last_name like ?
  )
`)
		args = append(args, search, search, search, search)
	}

	if input.AccountType != "" {
		query.WriteString("  and u.account_type = ?\n")
		args = append(args, input.AccountType)
	}

	if input.Cursor != nil {
		query.WriteString("  and u.id < ?\n")
		args = append(args, *input.Cursor)
	}

	query.WriteString("order by u.id desc\nlimit ?")
	args = append(args, input.Limit)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error in database.QueryContext: %w", err)
	}
	defer rows.Close()

	results := make([]*UserSummary, 0, input.Limit)
	for rows.Next() {
		summary := UserSummary{}
		var blockedAt sql.NullTime
		var aiAccountID sql.NullInt64
		if err := rows.Scan(
			&summary.ID,
			&summary.Email,
			&summary.Username,
			&summary.FirstName,
			&summary.LastName,
			&summary.About,
			&summary.AccountType,
			&blockedAt,
			&summary.CreatedAt,
			&summary.PostCount,
			&aiAccountID,
			&summary.AIEnabled, &summary.AINextPostAt, &summary.AILastPostAt,
			&summary.AIGenerationStatus, &summary.AIGenerationError,
			&summary.AIMinPostsPerDay, &summary.AIMaxPostsPerDay,
			&summary.AIContentMode, &summary.AICheckIntervalSeconds, &summary.AILastCheckedAt, &summary.AILastCheckOutcome, &summary.AIModelOptions,
		); err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		if blockedAt.Valid {
			value := blockedAt.Time
			summary.BlockedAt = &value
		}
		if aiAccountID.Valid {
			value := uint64(aiAccountID.Int64)
			summary.AIAccountID = &value
		}
		results = append(results, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uint64) (*UserSummary, error) {
	row := r.db.QueryRowContext(ctx, `
select
	u.id,
	u.email,
	u.username,
	u.first_name,
	u.last_name,
	u.about,
	u.account_type,
	u.blocked,
	u.created,
	(
		select count(*)
		from posts p
		where p.user_id = u.id
		  and p.deleted is null
	) as post_count,
	a.id as ai_account_id,
 COALESCE(a.enabled, false), a.next_generate_at, a.last_generated_at,
 COALESCE(a.generation_status, ''), COALESCE(a.generation_error, ''),
 COALESCE(a.min_posts_per_day, 0), COALESCE(a.max_posts_per_day, 0), COALESCE(a.content_mode,''),COALESCE(a.check_interval_seconds,0),a.last_checked_at,COALESCE(a.last_check_outcome,''),COALESCE(a.model_options,JSON_ARRAY())
from users u
left join ai_accounts a on a.user_id = u.id
where u.id = ?
  and u.deleted is null
`, userID)

	summary := UserSummary{}
	var blockedAt sql.NullTime
	var aiAccountID sql.NullInt64
	if err := row.Scan(
		&summary.ID,
		&summary.Email,
		&summary.Username,
		&summary.FirstName,
		&summary.LastName,
		&summary.About,
		&summary.AccountType,
		&blockedAt,
		&summary.CreatedAt,
		&summary.PostCount,
		&aiAccountID,
		&summary.AIEnabled, &summary.AINextPostAt, &summary.AILastPostAt,
		&summary.AIGenerationStatus, &summary.AIGenerationError,
		&summary.AIMinPostsPerDay, &summary.AIMaxPostsPerDay,
		&summary.AIContentMode, &summary.AICheckIntervalSeconds, &summary.AILastCheckedAt, &summary.AILastCheckOutcome, &summary.AIModelOptions,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error in database.QueryRowContext: %w", err)
	}
	if blockedAt.Valid {
		value := blockedAt.Time
		summary.BlockedAt = &value
	}
	if aiAccountID.Valid {
		value := uint64(aiAccountID.Int64)
		summary.AIAccountID = &value
	}

	return &summary, nil
}

func (r *Repository) BlockUser(ctx context.Context, userID uint64) error {
	if _, err := r.db.ExecContext(ctx, "update users set blocked=coalesce(blocked, utc_timestamp(6)) where id=? and deleted is null", userID); err != nil {
		return fmt.Errorf("error in database.ExecContext: %w", err)
	}

	return nil
}

func (r *Repository) ListPosts(ctx context.Context, input ListPostsInput) ([]*PostSummary, error) {
	args := make([]any, 0, 4)
	query := strings.Builder{}
	query.WriteString(`
select
	p.id,
	p.body,
	p.source,
	p.created,
	u.id,
	u.username,
	u.first_name,
	u.last_name,
	u.account_type
from posts p
join users u on u.id = p.user_id
where p.deleted is null
`)

	if input.UserID != nil {
		query.WriteString("  and p.user_id = ?\n")
		args = append(args, *input.UserID)
	}

	if input.Cursor != nil {
		query.WriteString("  and p.id < ?\n")
		args = append(args, *input.Cursor)
	}

	query.WriteString("order by p.id desc\nlimit ?")
	args = append(args, input.Limit)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error in database.QueryContext: %w", err)
	}
	defer rows.Close()

	results := make([]*PostSummary, 0, input.Limit)
	for rows.Next() {
		summary := PostSummary{}
		if err := rows.Scan(
			&summary.ID,
			&summary.Body,
			&summary.Source,
			&summary.CreatedAt,
			&summary.User.UserID,
			&summary.User.Username,
			&summary.User.FirstName,
			&summary.User.LastName,
			&summary.User.AccountType,
		); err != nil {
			return nil, fmt.Errorf("error in rows.Scan: %w", err)
		}
		results = append(results, &summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return results, nil
}

func (r *Repository) DeletePost(ctx context.Context, postID uint64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("error in database.BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, "update posts set deleted=utc_timestamp(6) where id=? and deleted is null", postID)
	if err != nil {
		return false, fmt.Errorf("error in database.ExecContext: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error in result.RowsAffected: %w", err)
	}

	if rowsAffected > 0 {
		if _, err := tx.ExecContext(ctx, "delete from post_bookmarks where post_id=?", postID); err != nil {
			return false, fmt.Errorf("error deleting post bookmarks: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("error in tx.Commit: %w", err)
	}

	return rowsAffected > 0, nil
}

func IsSupportedAccountType(accountType string) bool {
	switch accountType {
	case "", users.AccountTypeHuman, users.AccountTypeAI, users.AccountTypeSystem:
		return true
	default:
		return false
	}
}
