package aiposter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/posts"
)

type idGenerator interface {
	MustNext() uint64
}

type aiAccountsService interface {
	ClaimDueAIAccounts(ctx context.Context, now time.Time, limit int, staleAfter time.Duration) ([]*aiaccounts.AIAccount, error)
	CreatePostGenerationWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, generation *aiaccounts.PostGeneration) error
	FinishClaimWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, account *aiaccounts.AIAccount, now, next time.Time, success bool, message string) error
}

type postCreator interface {
	CreatePostWithExecutor(ctx context.Context, executor posts.CreateExecutor, input posts.CreatePostInput) (*posts.Post, error)
}

type transaction interface {
	posts.CreateExecutor
	aiaccounts.UpdateExecutor
	Commit() error
	Rollback() error
}

type transactionManager interface {
	Begin(ctx context.Context) (transaction, error)
}

type sqlTransactionManager struct {
	db *sql.DB
}

type sqlTransaction struct {
	tx *sql.Tx
}

type Runner struct {
	schedule          aiaccounts.Schedule
	logger            *zerolog.Logger
	transactions      transactionManager
	aiAccounts        aiAccountsService
	posts             postCreator
	generator         aigenerator.PostGenerator
	idGenerator       idGenerator
	pollInterval      time.Duration
	batchSize         int
	staleRunningAfter time.Duration
	now               func() time.Time
	rng               *rand.Rand
}

func NewSQLTransactionManager(db *sql.DB) transactionManager {
	return &sqlTransactionManager{db: db}
}

func (m *sqlTransactionManager) Begin(ctx context.Context) (transaction, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &sqlTransaction{tx: tx}, nil
}

func (t *sqlTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *sqlTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *sqlTransaction) Rollback() error {
	return t.tx.Rollback()
}

func NewRunner(
	logger *zerolog.Logger,
	transactions transactionManager,
	aiAccounts aiAccountsService,
	postCreator postCreator,
	generator aigenerator.PostGenerator,
	idGenerator idGenerator,
	pollInterval time.Duration,
	batchSize int,
	staleRunningAfter time.Duration,
) *Runner {
	if pollInterval <= 0 {
		pollInterval = time.Minute
	}
	if batchSize <= 0 {
		batchSize = 10
	}
	if staleRunningAfter <= 0 {
		staleRunningAfter = 15 * time.Minute
	}

	return &Runner{
		logger:            logger,
		transactions:      transactions,
		aiAccounts:        aiAccounts,
		posts:             postCreator,
		generator:         generator,
		idGenerator:       idGenerator,
		pollInterval:      pollInterval,
		batchSize:         batchSize,
		staleRunningAfter: staleRunningAfter,
		now:               time.Now,
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if err := r.RunOnce(ctx); err != nil {
			r.logError(err, "error processing ai poster batch")
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runner) SetSchedule(schedule aiaccounts.Schedule) { r.schedule = schedule }

func (r *Runner) RunOnce(ctx context.Context) error {
	// Claim just before processing: accounts never wait in a batch while their
	// leases expire. One bounded loop serves the whole roster, with no goroutine
	// or permanent process per identity.
	for i := 0; i < r.batchSize; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		claimed, err := r.aiAccounts.ClaimDueAIAccounts(ctx, r.now().UTC(), 1, r.staleRunningAfter)
		if err != nil {
			return err
		}
		if len(claimed) == 0 {
			break
		}
		for _, account := range claimed {
			if err := r.processAccount(ctx, account); err != nil {
				if errors.Is(err, aiaccounts.ErrClaimLost) {
					if r.logger != nil {
						r.logger.Info().Uint64("ai_account_id", account.ID).Msg("discarded result after AI account claim changed")
					}
				} else {
					r.logError(err, "error processing ai account", func(event *zerolog.Event) {
						event.Uint64("ai_account_id", account.ID).Uint64("user_id", account.UserID)
					})
				}
			}
		}
	}
	return nil
}

func (r *Runner) processAccount(ctx context.Context, account *aiaccounts.AIAccount) error {
	if account == nil || !account.Enabled || account.ClaimToken == "" {
		return aiaccounts.ErrClaimLost
	}
	if r.logger != nil {
		r.logger.Info().
			Uint64("ai_account_id", account.ID).
			Uint64("user_id", account.UserID).
			Str("topic", account.Topic).
			Msg("AI account claimed; generation started")
	}

	generationContext, cancel := context.WithTimeout(ctx, min(r.staleRunningAfter/2, 2*time.Minute))
	generated, err := r.generator.GeneratePost(generationContext, account)
	cancel()
	if err != nil {
		if errors.Is(err, aigenerator.ErrRejected) {
			return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusRejected, err.Error(), "")
		}
		return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusFailed, "", err.Error())
	}

	candidateBody := strings.TrimSpace(generated.Body)
	if candidateBody == "" || len(candidateBody) > posts.MaxBodyBytes {
		return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusRejected, "invalid generated post body", "")
	}

	tx, err := r.transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin finalize transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	now := r.now().UTC()
	nextGenerateAt := r.schedule.Next(now, account.MinPostsPerDay, account.MaxPostsPerDay)
	if err := r.aiAccounts.FinishClaimWithExecutor(ctx, tx, account, now, nextGenerateAt, true, ""); err != nil {
		return err
	}
	if r.logger != nil {
		r.logger.Info().Uint64("ai_account_id", account.ID).Str("model", generated.Model).Msg("AI generation succeeded")
	}
	post, err := r.posts.CreatePostWithExecutor(ctx, tx, posts.CreatePostInput{
		AuthorID: account.UserID,
		Body:     candidateBody,
		Source:   posts.SourceAI,
	})
	if err != nil {
		_ = tx.Rollback()
		return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusFailed, "", err.Error())
	}

	postID := post.ID
	generation := &aiaccounts.PostGeneration{
		ID:            r.idGenerator.MustNext(),
		AIAccountID:   account.ID,
		PostID:        &postID,
		Status:        aiaccounts.PostGenerationStatusPosted,
		Prompt:        generated.Prompt,
		CandidateBody: candidateBody,
		FinalBody:     post.Body,
		Model:         generated.Model,
	}

	if err := r.aiAccounts.CreatePostGenerationWithExecutor(ctx, tx, generation); err != nil {
		_ = tx.Rollback()
		return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusFailed, "", err.Error())
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return r.recordFailure(ctx, account, generated, aiaccounts.PostGenerationStatusFailed, "", err.Error())
	}
	committed = true

	if r.logger != nil {
		r.logger.Info().
			Uint64("ai_account_id", account.ID).
			Uint64("user_id", account.UserID).
			Uint64("post_id", post.ID).
			Str("topic", account.Topic).
			Time("next_generate_at", nextGenerateAt).
			Msg("posted ai account content")
	}
	return nil
}

func (r *Runner) recordFailure(ctx context.Context, account *aiaccounts.AIAccount, generated aigenerator.GeneratedPost, status string, rejectReason string, generationErr string) error {
	tx, err := r.transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin failure transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	generation := &aiaccounts.PostGeneration{
		ID:            r.idGenerator.MustNext(),
		AIAccountID:   account.ID,
		Status:        status,
		Prompt:        generated.Prompt,
		CandidateBody: strings.TrimSpace(generated.Body),
		RejectReason:  rejectReason,
		Error:         generationErr,
		Model:         generated.Model,
	}
	accountError := generationErr
	if accountError == "" {
		accountError = rejectReason
	}
	if accountError == "" {
		accountError = "generation failed"
	}

	nextRetry := computeRetryAtWithRand(r.now().UTC(), r.rng, account.ConsecutiveFailures)
	if err := r.aiAccounts.FinishClaimWithExecutor(ctx, tx, account, r.now().UTC(), nextRetry, false, accountError); err != nil {
		return err
	}
	if err := r.aiAccounts.CreatePostGenerationWithExecutor(ctx, tx, generation); err != nil {
		return fmt.Errorf("record failure generation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failure transaction: %w", err)
	}
	committed = true

	if status == aiaccounts.PostGenerationStatusRejected {
		if r.logger != nil {
			r.logger.Warn().
				Uint64("ai_account_id", account.ID).
				Uint64("user_id", account.UserID).
				Str("reason", accountError).
				Time("next_generate_at", nextRetry).
				Msg("rejected ai account content")
		}
	} else {
		r.logError(errors.New(accountError), "ai account generation failed", func(event *zerolog.Event) {
			event.Uint64("ai_account_id", account.ID).
				Uint64("user_id", account.UserID).
				Str("reason", accountError).Time("next_generate_at", nextRetry)
		})
	}

	return nil
}

func (r *Runner) logError(err error, message string, enrichers ...func(event *zerolog.Event)) {
	if r.logger == nil {
		return
	}

	event := r.logger.Error().Err(err)
	for _, enrich := range enrichers {
		if enrich != nil {
			enrich(event)
		}
	}
	event.Msg(message)
}
