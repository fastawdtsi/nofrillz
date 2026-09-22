package aiaccounts

import (
	"context"
	"errors"
	"time"
)

var ErrClaimLost = errors.New("AI account claim expired, was cancelled, or is already being processed")

// FinishClaimWithExecutor must run BEFORE inserting a post in the SAME
// transaction. The conditional update locks/fences the account until commit;
// rollback restores the claim if post creation fails.
func (r *Repository) FinishClaimWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount, now, next time.Time, success bool, message string) error {
	if account == nil || account.ClaimToken == "" {
		return ErrClaimLost
	}
	var last any
	if success {
		last = now.UTC()
	}
	result, err := executor.ExecContext(ctx, `
		UPDATE ai_accounts SET generation_status='idle', claim_token=NULL,
		generation_started_at=NULL, next_generate_at=?, generation_error=?,
		last_generated_at=COALESCE(?, last_generated_at),
		consecutive_failures=IF(?, 0, consecutive_failures+1)
		WHERE id=? AND claim_token=? AND generation_status='running' AND enabled=TRUE
		AND EXISTS (SELECT 1 FROM users u WHERE u.id=ai_accounts.user_id AND u.deleted IS NULL AND u.blocked IS NULL)`,
		next.UTC(), nullIfEmpty(message), last, success, account.ID, account.ClaimToken)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrClaimLost
	}
	return nil
}

func (s *Service) FinishClaimWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount, now, next time.Time, success bool, message string) error {
	return s.repository.FinishClaimWithExecutor(ctx, executor, account, now, next, success, message)
}

func (r *Repository) UpdateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error {
	_, err := executor.ExecContext(ctx, `UPDATE ai_accounts SET enabled=?, topic=?, description=?, system_prompt=?, style_prompt=?,
		min_posts_per_day=?, max_posts_per_day=?, next_generate_at=?, generation_status='idle',
		claim_token=NULL, generation_started_at=NULL, generation_error=NULL, consecutive_failures=0 WHERE id=?`,
		account.Enabled, account.Topic, account.Description, account.SystemPrompt, account.StylePrompt,
		account.MinPostsPerDay, account.MaxPostsPerDay, account.NextGenerateAt, account.ID)
	return err
}

func (s *Service) UpdateWithExecutor(ctx context.Context, executor UpdateExecutor, account *AIAccount) error {
	return s.repository.UpdateWithExecutor(ctx, executor, account)
}
