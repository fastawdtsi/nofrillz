package aiposter

import (
	"context"
	"errors"
	"nofrillz/internal/aiaccounts"
	"testing"
	"time"
)

type testClaims struct {
	rows   []*aiaccounts.AIAccount
	limits []int
}

func (c *testClaims) ClaimDueAIAccounts(_ context.Context, _ time.Time, limit int, _ time.Duration) ([]*aiaccounts.AIAccount, error) {
	c.limits = append(c.limits, limit)
	if len(c.rows) == 0 {
		return nil, nil
	}
	r := c.rows[:1]
	c.rows = c.rows[1:]
	return r, nil
}

type testProcessor struct{ seen []uint64 }

func (p *testProcessor) Process(_ context.Context, a *aiaccounts.AIAccount) error {
	p.seen = append(p.seen, a.ID)
	if a.ID == 1 {
		return errors.New("one account failed")
	}
	return nil
}
func TestRunnerSkipsDisabledAndContinuesAfterFailure(t *testing.T) {
	claims := &testClaims{rows: []*aiaccounts.AIAccount{{ID: 1, Enabled: true, ClaimToken: "a"}, {ID: 2, Enabled: false, ClaimToken: "b"}, {ID: 3, Enabled: true, ClaimToken: "c"}}}
	processor := &testProcessor{}
	r := NewRunner(nil, claims, processor, time.Minute, 10, 15*time.Minute)
	if err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(processor.seen) != 2 || processor.seen[0] != 1 || processor.seen[1] != 3 {
		t.Fatalf("bad processing: %v", processor.seen)
	}
	for _, limit := range claims.limits {
		if limit != 1 {
			t.Fatal("claims must not wait in a batch")
		}
	}
}
func TestRunnerHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := NewRunner(nil, &testClaims{}, &testProcessor{}, time.Minute, 10, time.Minute)
	if !errors.Is(r.RunOnce(ctx), context.Canceled) {
		t.Fatal("cancel ignored")
	}
}
