package sessions

import (
	"context"
	"errors"
	"testing"
	"time"
)

type sessionsRepoStub struct {
	createErr    error
	getActive    *Session
	getActiveErr error
	revokeOK     bool
	revokeErr    error

	createRefreshErr    error
	getActiveRefresh    *RefreshToken
	getActiveRefreshErr error
	revokeRefreshOK     bool
	revokeRefreshErr    error
	revokeBySessionOK   bool
	revokeBySessionErr  error

	createSessionID uint64
	createUserID    uint64
	getSessionID    uint64
	revokeSessionID uint64

	createRefreshTokenHash string
	createRefreshSessionID uint64
	createRefreshExpiresAt time.Time
	getRefreshTokenHash    string
	revokeRefreshTokenHash string
	revokeBySessionID      uint64
}

func (s *sessionsRepoStub) Create(ctx context.Context, sessionID uint64, userID uint64) error {
	s.createSessionID = sessionID
	s.createUserID = userID
	return s.createErr
}

func (s *sessionsRepoStub) GetActiveByID(ctx context.Context, sessionID uint64) (*Session, error) {
	s.getSessionID = sessionID
	return s.getActive, s.getActiveErr
}

func (s *sessionsRepoStub) RevokeByID(ctx context.Context, sessionID uint64) (bool, error) {
	s.revokeSessionID = sessionID
	return s.revokeOK, s.revokeErr
}

func (s *sessionsRepoStub) CreateRefresh(ctx context.Context, tokenHash string, sessionID uint64, expiresAt time.Time) error {
	s.createRefreshTokenHash = tokenHash
	s.createRefreshSessionID = sessionID
	s.createRefreshExpiresAt = expiresAt
	return s.createRefreshErr
}

func (s *sessionsRepoStub) GetActiveRefreshByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	s.getRefreshTokenHash = tokenHash
	return s.getActiveRefresh, s.getActiveRefreshErr
}

func (s *sessionsRepoStub) RevokeRefreshByTokenHash(ctx context.Context, tokenHash string) (bool, error) {
	s.revokeRefreshTokenHash = tokenHash
	return s.revokeRefreshOK, s.revokeRefreshErr
}

func (s *sessionsRepoStub) RevokeRefreshBySessionID(ctx context.Context, sessionID uint64) (bool, error) {
	s.revokeBySessionID = sessionID
	return s.revokeBySessionOK, s.revokeBySessionErr
}

func TestServiceCreate(t *testing.T) {
	expectedErr := errors.New("create error")
	repo := &sessionsRepoStub{createErr: expectedErr}
	service := NewService(repo)

	err := service.Create(context.Background(), 11, 22)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected create error, got %v", err)
	}
	if repo.createSessionID != 11 || repo.createUserID != 22 {
		t.Fatalf("expected create args to be forwarded")
	}
}

func TestServiceGetActiveByID(t *testing.T) {
	expected := &Session{ID: 9}
	repo := &sessionsRepoStub{getActive: expected}
	service := NewService(repo)

	got, err := service.GetActiveByID(context.Background(), 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same session pointer from repository")
	}
	if repo.getSessionID != 9 {
		t.Fatalf("expected session ID argument to be forwarded")
	}
}

func TestServiceRevokeByID(t *testing.T) {
	repo := &sessionsRepoStub{revokeOK: true}
	service := NewService(repo)

	ok, err := service.RevokeByID(context.Background(), 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected revoke to return true")
	}
	if repo.revokeSessionID != 12 {
		t.Fatalf("expected session ID argument to be forwarded")
	}
}

func TestServiceCreateRefresh(t *testing.T) {
	repo := &sessionsRepoStub{}
	service := NewService(repo)
	expiresAt := time.Now().UTC().Add(2 * time.Hour)

	err := service.CreateRefresh(context.Background(), "hash-a", 14, expiresAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createRefreshTokenHash != "hash-a" || repo.createRefreshSessionID != 14 {
		t.Fatalf("expected create refresh args to be forwarded")
	}
}

func TestServiceGetActiveRefreshByTokenHash(t *testing.T) {
	expected := &RefreshToken{TokenHash: "hash-b"}
	repo := &sessionsRepoStub{getActiveRefresh: expected}
	service := NewService(repo)

	got, err := service.GetActiveRefreshByTokenHash(context.Background(), "hash-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same refresh token pointer from repository")
	}
	if repo.getRefreshTokenHash != "hash-b" {
		t.Fatalf("expected refresh token hash argument to be forwarded")
	}
}

func TestServiceRevokeRefreshByTokenHash(t *testing.T) {
	repo := &sessionsRepoStub{revokeRefreshOK: true}
	service := NewService(repo)

	ok, err := service.RevokeRefreshByTokenHash(context.Background(), "hash-c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected revoke refresh to return true")
	}
	if repo.revokeRefreshTokenHash != "hash-c" {
		t.Fatalf("expected refresh token hash argument to be forwarded")
	}
}

func TestServiceRevokeRefreshBySessionID(t *testing.T) {
	repo := &sessionsRepoStub{revokeBySessionOK: true}
	service := NewService(repo)

	ok, err := service.RevokeRefreshBySessionID(context.Background(), 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected revoke refresh by session to return true")
	}
	if repo.revokeBySessionID != 16 {
		t.Fatalf("expected session ID argument to be forwarded")
	}
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	token, err := GenerateAccessToken("test-secret", 25, 30, time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	claims, err := ParseAccessToken("test-secret", token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.UserID != 25 || claims.SessionID != 30 {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestHashToken(t *testing.T) {
	hashA := HashToken("abc")
	hashB := HashToken("abc")
	if hashA != hashB {
		t.Fatalf("expected deterministic hash")
	}
	if len(hashA) != 64 {
		t.Fatalf("expected 64-char hex hash")
	}
}
