package aiposter

import (
	"context"
	"errors"
	"github.com/rs/zerolog"
	"nofrillz/internal/aiaccounts"
	"time"
)

type accountClaimer interface {
	ClaimDueAIAccounts(context.Context, time.Time, int, time.Duration) ([]*aiaccounts.AIAccount, error)
}
type contentProcessor interface {
	Process(context.Context, *aiaccounts.AIAccount) error
}
type Runner struct {
	logger            *zerolog.Logger
	aiAccounts        accountClaimer
	processor         contentProcessor
	pollInterval      time.Duration
	batchSize         int
	staleRunningAfter time.Duration
	now               func() time.Time
}

func NewRunner(logger *zerolog.Logger, accounts accountClaimer, processor contentProcessor, poll time.Duration, batch int, stale time.Duration) *Runner {
	if poll <= 0 {
		poll = time.Minute
	}
	if batch <= 0 {
		batch = 10
	}
	if stale <= 0 {
		stale = 15 * time.Minute
	}
	return &Runner{logger, accounts, processor, poll, batch, stale, time.Now}
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

func (r *Runner) processAccount(ctx context.Context, a *aiaccounts.AIAccount) error {
	if a == nil || !a.Enabled || a.ClaimToken == "" {
		return aiaccounts.ErrClaimLost
	}
	workCtx, cancel := context.WithTimeout(ctx, min(r.staleRunningAfter/2, 5*time.Minute))
	defer cancel()
	return r.processor.Process(workCtx, a)
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
