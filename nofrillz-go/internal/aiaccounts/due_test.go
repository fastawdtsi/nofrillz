package aiaccounts

import (
	"testing"
	"time"
)

func TestIsDueForClaimIdleDue(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	next := now.Add(-time.Minute)
	account := &AIAccount{
		Enabled:          true,
		NextGenerateAt:   &next,
		GenerationStatus: GenerationStatusIdle,
	}

	if !IsDueForClaim(account, now, 15*time.Minute) {
		t.Fatalf("expected idle due account to be claimable")
	}
}

func TestIsDueForClaimDisabledNotClaimed(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	next := now.Add(-time.Minute)
	account := &AIAccount{
		Enabled:          false,
		NextGenerateAt:   &next,
		GenerationStatus: GenerationStatusIdle,
	}

	if IsDueForClaim(account, now, 15*time.Minute) {
		t.Fatalf("expected disabled account to be skipped")
	}
}

func TestIsDueForClaimStaleRunningReclaimed(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	next := now.Add(-time.Minute)
	started := now.Add(-16 * time.Minute)
	account := &AIAccount{
		Enabled:             true,
		NextGenerateAt:      &next,
		GenerationStatus:    GenerationStatusRunning,
		GenerationStartedAt: &started,
	}

	if !IsDueForClaim(account, now, 15*time.Minute) {
		t.Fatalf("expected stale running account to be reclaimable")
	}
}

func TestIsDueForClaimFreshRunningSkipped(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	next := now.Add(-time.Minute)
	started := now.Add(-5 * time.Minute)
	account := &AIAccount{
		Enabled:             true,
		NextGenerateAt:      &next,
		GenerationStatus:    GenerationStatusRunning,
		GenerationStartedAt: &started,
	}

	if IsDueForClaim(account, now, 15*time.Minute) {
		t.Fatalf("expected fresh running account to be skipped")
	}
}
