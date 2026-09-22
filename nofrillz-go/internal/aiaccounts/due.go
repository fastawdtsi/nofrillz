package aiaccounts

import "time"

func IsDueForClaim(account *AIAccount, now time.Time, staleAfter time.Duration) bool {
	if account == nil || !account.Enabled || account.NextGenerateAt == nil {
		return false
	}
	if account.NextGenerateAt.After(now.UTC()) {
		return false
	}

	switch account.GenerationStatus {
	case "", GenerationStatusIdle:
		return true
	case GenerationStatusRunning:
		if account.GenerationStartedAt == nil {
			return true
		}
		return !account.GenerationStartedAt.After(now.UTC().Add(-staleAfter))
	default:
		return false
	}
}
