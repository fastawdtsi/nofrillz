package apns

import (
	"context"
	"errors"
	"testing"
)

type apnsRepoStub struct {
	tokens       []*DeviceToken
	listErr      error
	upsertUserID uint64
	upsertToken  string
	upsertErr    error
	deleted      []string
	deleteErr    error
}

func (s *apnsRepoStub) ListByUserID(ctx context.Context, userID uint64) ([]*DeviceToken, error) {
	return s.tokens, s.listErr
}

func (s *apnsRepoStub) Upsert(ctx context.Context, userID uint64, token string) error {
	s.upsertUserID = userID
	s.upsertToken = token
	return s.upsertErr
}

func (s *apnsRepoStub) DeleteTokens(ctx context.Context, tokens []string) error {
	s.deleted = append([]string{}, tokens...)
	return s.deleteErr
}

type apnsProviderStub struct {
	results map[string]*DeliveryResult
	errs    map[string]error
	sent    []string
}

func (s *apnsProviderStub) Send(ctx context.Context, deviceToken string, title string, body string) (*DeliveryResult, error) {
	s.sent = append(s.sent, deviceToken)
	return s.results[deviceToken], s.errs[deviceToken]
}

func TestRegisterTokenNormalizesAndStores(t *testing.T) {
	repo := &apnsRepoStub{}
	service := NewService(nil, repo, noopProvider{})

	token, err := service.RegisterToken(context.Background(), 11, " AABBCCDD ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "aabbccdd" {
		t.Fatalf("expected normalized token, got %q", token)
	}
	if repo.upsertUserID != 11 || repo.upsertToken != "aabbccdd" {
		t.Fatalf("expected repository upsert to be called with normalized token")
	}
}

func TestSendOneOffSendsAllTokensAndPrunesInvalidOnes(t *testing.T) {
	repo := &apnsRepoStub{
		tokens: []*DeviceToken{
			{Token: "aa"},
			{Token: "bb"},
			{Token: "cc"},
		},
	}
	provider := &apnsProviderStub{
		results: map[string]*DeliveryResult{
			"bb": {Reason: "Unregistered", InvalidToken: true},
		},
		errs: map[string]error{
			"bb": errors.New("apns rejected token"),
		},
	}
	service := NewService(nil, repo, provider)

	summary, err := service.SendOneOff(context.Background(), 22, "Hello", "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Total != 3 || summary.Success != 2 || summary.Invalid != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(provider.sent) != 3 {
		t.Fatalf("expected provider to receive all tokens, got %d", len(provider.sent))
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "bb" {
		t.Fatalf("expected invalid token to be pruned, got %#v", repo.deleted)
	}
}

func TestSendOneOffReturnsErrorForTransportFailures(t *testing.T) {
	repo := &apnsRepoStub{tokens: []*DeviceToken{{Token: "aa"}}}
	provider := &apnsProviderStub{
		errs: map[string]error{"aa": errors.New("network down")},
	}
	service := NewService(nil, repo, provider)

	summary, err := service.SendOneOff(context.Background(), 1, "Hello", "World")
	if err == nil {
		t.Fatalf("expected error")
	}
	if summary == nil || summary.Failed != 1 {
		t.Fatalf("expected failed summary, got %+v", summary)
	}
}
