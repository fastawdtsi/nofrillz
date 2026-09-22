package aiaccounts

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type UpdateExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, account *AIAccount) error {
	return r.CreateWithExecutor(ctx, r.db, account)
}

func (r *Repository) CreateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error {
	var (
		description         any
		stylePrompt         any
		nextGenerateAt      any
		lastGeneratedAt     any
		generationStartedAt any
		generationError     any
	)

	description = nullIfEmpty(account.Description)
	stylePrompt = nullIfEmpty(account.StylePrompt)
	generationError = nullIfEmpty(account.GenerationError)
	if account.NextGenerateAt != nil {
		nextGenerateAt = account.NextGenerateAt.UTC()
	}
	if account.LastGeneratedAt != nil {
		lastGeneratedAt = account.LastGeneratedAt.UTC()
	}
	if account.GenerationStartedAt != nil {
		generationStartedAt = account.GenerationStartedAt.UTC()
	}

	_, err := executor.ExecContext(
		ctx,
		insertAIAccountQuery,
		account.ID,
		account.UserID,
		account.Enabled,
		account.Topic,
		description,
		account.SystemPrompt,
		stylePrompt,
		account.MinPostsPerDay,
		account.MaxPostsPerDay,
		nextGenerateAt,
		lastGeneratedAt,
		account.GenerationStatus,
		generationStartedAt,
		generationError,
	)
	if err != nil {
		return fmt.Errorf("error in database.ExecContext: %w", err)
	}

	return r.saveContentConfig(ctx, executor, account)
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*AIAccount, error) {
	row := r.db.QueryRowContext(ctx, selectAIAccountByIDQuery, id)

	account, err := scanAIAccount(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return account, nil
}

func (r *Repository) ListPostGenerationsByAccountID(ctx context.Context, accountID uint64, limit int) ([]*PostGeneration, error) {
	if limit <= 0 {
		return []*PostGeneration{}, nil
	}

	rows, err := r.db.QueryContext(ctx, selectPostGenerationsByAccountIDQuery, accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("error in database.QueryContext: %w", err)
	}
	defer rows.Close()

	generations := make([]*PostGeneration, 0, limit)
	for rows.Next() {
		generation, err := scanPostGeneration(rows)
		if err != nil {
			return nil, err
		}
		generations = append(generations, generation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	return generations, nil
}

func (r *Repository) CreatePostGeneration(ctx context.Context, generation *PostGeneration) error {
	return r.CreatePostGenerationWithExecutor(ctx, r.db, generation)
}

func (r *Repository) ClaimDueAIAccounts(ctx context.Context, now time.Time, limit int, staleAfter time.Duration) ([]*AIAccount, error) {
	if limit <= 0 {
		return []*AIAccount{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error in database.BeginTx: %w", err)
	}
	defer tx.Rollback()

	now = now.UTC()
	staleBefore := now.Add(-staleAfter)

	rows, err := tx.QueryContext(ctx, selectDueAccountsForClaimQuery, now, staleBefore, limit)
	if err != nil {
		return nil, fmt.Errorf("error in database.QueryContext: %w", err)
	}
	defer rows.Close()

	accounts := make([]*AIAccount, 0, limit)

	for rows.Next() {
		account, err := scanAIAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error in rows.Err: %w", err)
	}

	if len(accounts) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("error in tx.Commit: %w", err)
		}
		return accounts, nil
	}

	// Each claim has a new fencing token. A stale worker cannot finalize a
	// reclaimed account, even if it returns after the new owner has published.
	for _, account := range accounts {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return nil, err
		}
		account.ClaimToken = hex.EncodeToString(token[:])
		if _, err := tx.ExecContext(ctx, "update ai_accounts set generation_status='running', generation_started_at=?, claim_token=?, generation_error=null where id=?", now, account.ClaimToken, account.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error in tx.Commit: %w", err)
	}

	for _, account := range accounts {
		account.GenerationStatus = GenerationStatusRunning
		startedAt := now
		account.GenerationStartedAt = &startedAt
		account.GenerationError = ""
	}

	return accounts, nil
}

func (r *Repository) CreatePostGenerationWithExecutor(ctx context.Context, executor UpdateExecutor, generation *PostGeneration) error {
	var postID any
	if generation.PostID != nil {
		postID = *generation.PostID
	}

	_, err := executor.ExecContext(
		ctx,
		insertPostGenerationQuery,
		generation.ID,
		generation.AIAccountID,
		postID,
		generation.Status,
		nullIfEmpty(generation.Prompt),
		nullIfEmpty(generation.CandidateBody),
		nullIfEmpty(generation.FinalBody),
		nullIfEmpty(generation.RejectReason),
		nullIfEmpty(generation.Error),
		nullIfEmpty(generation.Model),
	)
	if err != nil {
		return fmt.Errorf("error in database.ExecContext: %w", err)
	}

	return nil
}

func (r *Repository) MarkGenerationSuccessWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error {
	result, err := executor.ExecContext(ctx, updateAccountSuccessQuery, nextGenerateAt.UTC(), lastGeneratedAt.UTC(), accountID)
	if err != nil {
		return fmt.Errorf("error in database.ExecContext: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrClaimLost
	}
	return nil
}

func (r *Repository) MarkGenerationFailureWithExecutor(ctx context.Context, executor UpdateExecutor, accountID uint64, nextGenerateAt time.Time, generationError string) error {
	_, err := executor.ExecContext(ctx, updateAccountFailureQuery, nextGenerateAt.UTC(), generationError, accountID)
	if err != nil {
		return fmt.Errorf("error in database.ExecContext: %w", err)
	}

	return nil
}

func scanAIAccount(scanner interface {
	Scan(dest ...any) error
}) (*AIAccount, error) {
	account := AIAccount{}
	var description sql.NullString
	var stylePrompt sql.NullString
	var nextGenerateAt sql.NullTime
	var lastGeneratedAt sql.NullTime
	var generationStartedAt sql.NullTime
	var generationError sql.NullString
	var claimToken sql.NullString
	var sourceURLs, modelOptions []byte

	err := scanner.Scan(
		&account.ID,
		&account.UserID,
		&account.Enabled,
		&account.Topic,
		&description,
		&account.SystemPrompt,
		&stylePrompt,
		&account.MinPostsPerDay,
		&account.MaxPostsPerDay,
		&nextGenerateAt,
		&lastGeneratedAt,
		&account.GenerationStatus,
		&generationStartedAt,
		&generationError,
		&claimToken,
		&account.ConsecutiveFailures,
		&account.ContentMode, &account.CheckIntervalSeconds, &account.Exclusions,
		&sourceURLs, &modelOptions, &account.DefaultModelOption, &account.SourceMaxAgeHours,
		&account.LastCheckedAt, &account.LastCheckOutcome,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error in rows.Scan: %w", err)
	}

	if len(sourceURLs) > 0 {
		if err := json.Unmarshal(sourceURLs, &account.SourceURLs); err != nil {
			return nil, err
		}
	}
	if len(modelOptions) > 0 {
		if err := json.Unmarshal(modelOptions, &account.ModelOptions); err != nil {
			return nil, err
		}
	}
	account.Description = description.String
	account.StylePrompt = stylePrompt.String
	account.GenerationError = generationError.String
	account.ClaimToken = claimToken.String
	if nextGenerateAt.Valid {
		value := nextGenerateAt.Time
		account.NextGenerateAt = &value
	}
	if lastGeneratedAt.Valid {
		value := lastGeneratedAt.Time
		account.LastGeneratedAt = &value
	}
	if generationStartedAt.Valid {
		value := generationStartedAt.Time
		account.GenerationStartedAt = &value
	}

	return &account, nil
}

func scanPostGeneration(scanner interface {
	Scan(dest ...any) error
}) (*PostGeneration, error) {
	generation := PostGeneration{}
	var (
		postID        sql.NullInt64
		prompt        sql.NullString
		candidateBody sql.NullString
		finalBody     sql.NullString
		rejectReason  sql.NullString
		errText       sql.NullString
		model         sql.NullString
	)

	err := scanner.Scan(
		&generation.ID,
		&generation.AIAccountID,
		&postID,
		&generation.Status,
		&prompt,
		&candidateBody,
		&finalBody,
		&rejectReason,
		&errText,
		&model,
		&generation.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	generation.Prompt = prompt.String
	generation.CandidateBody = candidateBody.String
	generation.FinalBody = finalBody.String
	generation.RejectReason = rejectReason.String
	generation.Error = errText.String
	generation.Model = model.String
	if postID.Valid {
		value := uint64(postID.Int64)
		generation.PostID = &value
	}

	return &generation, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}

	return value
}
