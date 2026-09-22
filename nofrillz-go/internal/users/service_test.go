package users

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

type usersRepoStub struct {
	createErr                 error
	getByID                   *User
	getByIDErr                error
	getByIDForRequester       *User
	getByIDForRequesterErr    error
	getByEmail                *User
	getByEmailErr             error
	searchResults             []*User
	searchErr                 error
	searchForRequesterResults []*User
	searchForRequesterErr     error
	followersPageResults      []*User
	followersPageErr          error

	created                           *User
	idArg                             uint64
	getByIDForRequesterIDArg          uint64
	getByIDForRequesterRequesterIDArg uint64
	emailArg                          string
	searchPrefixArg                   string
	searchLimitArg                    int
	searchForRequesterPrefixArg       string
	searchForRequesterLimitArg        int
	searchForRequesterRequesterIDArg  uint64
	followersPageUserIDArg            uint64
	followersPageRequesterIDArg       uint64
	followersPageCursorArg            *uint64
	followersPageLimitArg             int
}

func (s *usersRepoStub) Create(ctx context.Context, user *User) error {
	s.created = user
	return s.createErr
}

func (s *usersRepoStub) CreateWithExecutor(ctx context.Context, executor CreateExecutor, user *User) error {
	s.created = user
	return s.createErr
}

type usersExecutorStub struct{}

func (s *usersExecutorStub) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (s *usersRepoStub) GetByID(ctx context.Context, ID uint64) (*User, error) {
	s.idArg = ID
	return s.getByID, s.getByIDErr
}

func (s *usersRepoStub) GetByIDForRequester(ctx context.Context, ID uint64, requesterUserID uint64) (*User, error) {
	s.getByIDForRequesterIDArg = ID
	s.getByIDForRequesterRequesterIDArg = requesterUserID
	return s.getByIDForRequester, s.getByIDForRequesterErr
}

func (s *usersRepoStub) GetByEmail(ctx context.Context, email string) (*User, error) {
	s.emailArg = email
	return s.getByEmail, s.getByEmailErr
}

func (s *usersRepoStub) SearchByUsernamePrefix(ctx context.Context, prefix string, limit int) ([]*User, error) {
	s.searchPrefixArg = prefix
	s.searchLimitArg = limit
	return s.searchResults, s.searchErr
}

func (s *usersRepoStub) SearchByUsernamePrefixForRequester(ctx context.Context, prefix string, requesterUserID uint64, limit int) ([]*User, error) {
	s.searchForRequesterPrefixArg = prefix
	s.searchForRequesterRequesterIDArg = requesterUserID
	s.searchForRequesterLimitArg = limit
	return s.searchForRequesterResults, s.searchForRequesterErr
}

func (s *usersRepoStub) ListFollowersPageForRequester(ctx context.Context, userID uint64, requesterUserID uint64, cursor *uint64, limit int) ([]*User, error) {
	s.followersPageUserIDArg = userID
	s.followersPageRequesterIDArg = requesterUserID
	s.followersPageCursorArg = cursor
	s.followersPageLimitArg = limit
	return s.followersPageResults, s.followersPageErr
}

func TestServiceCreate(t *testing.T) {
	expectedErr := errors.New("create error")
	repo := &usersRepoStub{createErr: expectedErr}
	service := NewService(repo)
	user := &User{ID: 1, Email: "a@example.com"}

	err := service.Create(context.Background(), user)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected create error, got %v", err)
	}
	if repo.created != user {
		t.Fatalf("expected repository to receive user pointer")
	}
}

func TestServiceCreateWithExecutor(t *testing.T) {
	repo := &usersRepoStub{}
	service := NewService(repo)
	user := &User{ID: 1, Email: "a@example.com"}

	if err := service.CreateWithExecutor(context.Background(), &usersExecutorStub{}, user); err != nil {
		t.Fatalf("CreateWithExecutor: %v", err)
	}
	if repo.created != user {
		t.Fatalf("expected repository to receive user pointer")
	}
}

func TestServiceGetByID(t *testing.T) {
	expected := &User{ID: 42}
	repo := &usersRepoStub{getByID: expected}
	service := NewService(repo)

	got, err := service.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same user pointer from repository")
	}
	if repo.idArg != 42 {
		t.Fatalf("expected ID argument to be forwarded")
	}
}

func TestServiceGetByIDForRequester(t *testing.T) {
	expected := &User{ID: 42}
	repo := &usersRepoStub{getByIDForRequester: expected}
	service := NewService(repo)

	got, err := service.GetByIDForRequester(context.Background(), 42, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same user pointer from repository")
	}
	if repo.getByIDForRequesterIDArg != 42 || repo.getByIDForRequesterRequesterIDArg != 99 {
		t.Fatalf("expected ID/requester arguments to be forwarded")
	}
}

func TestServiceGetByEmail(t *testing.T) {
	expected := &User{Email: "a@example.com"}
	repo := &usersRepoStub{getByEmail: expected}
	service := NewService(repo)

	got, err := service.GetByEmail(context.Background(), "a@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected same user pointer from repository")
	}
	if repo.emailArg != "a@example.com" {
		t.Fatalf("expected email argument to be forwarded")
	}
}

func TestServiceSearchByUsernamePrefix(t *testing.T) {
	expected := []*User{{ID: 10, Username: "alice"}}
	repo := &usersRepoStub{searchResults: expected}
	service := NewService(repo)

	got, err := service.SearchByUsernamePrefix(context.Background(), "ali", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != expected[0] {
		t.Fatalf("unexpected search results")
	}
	if repo.searchPrefixArg != "ali" || repo.searchLimitArg != 5 {
		t.Fatalf("expected search args to be forwarded")
	}
}

func TestServiceSearchByUsernamePrefixForRequester(t *testing.T) {
	expected := []*User{{ID: 10, Username: "alice"}}
	repo := &usersRepoStub{searchForRequesterResults: expected}
	service := NewService(repo)

	got, err := service.SearchByUsernamePrefixForRequester(context.Background(), "ali", 44, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != expected[0] {
		t.Fatalf("unexpected search results")
	}
	if repo.searchForRequesterPrefixArg != "ali" || repo.searchForRequesterRequesterIDArg != 44 || repo.searchForRequesterLimitArg != 5 {
		t.Fatalf("expected search args to be forwarded")
	}
}

func TestServiceListFollowersPageForRequester(t *testing.T) {
	page := []*User{
		{ID: 10, Username: "u10"},
		{ID: 9, Username: "u9"},
		{ID: 8, Username: "u8"},
	}
	repo := &usersRepoStub{followersPageResults: page}
	service := NewService(repo)

	got, next, err := service.ListFollowersPageForRequester(context.Background(), 20, 21, nil, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 followers, got %d", len(got))
	}
	if next == nil || *next != 9 {
		t.Fatalf("expected next cursor to point to last returned follower")
	}
	if repo.followersPageUserIDArg != 20 || repo.followersPageRequesterIDArg != 21 || repo.followersPageLimitArg != 3 {
		t.Fatalf("expected paginated list args to be forwarded with limit+1")
	}
}

func TestServiceUnicodeRoundTrip(t *testing.T) {
	unicodeUsername := "مستخدم🌍"
	unicodePrefix := "مست"

	repo := &usersRepoStub{
		getByIDForRequester:       &User{ID: 7, Username: unicodeUsername},
		searchForRequesterResults: []*User{{ID: 8, Username: unicodeUsername}},
		followersPageResults:      []*User{{ID: 9, Username: unicodeUsername}},
	}
	service := NewService(repo)

	user := &User{ID: 6, Email: "unicode@example.com", Username: unicodeUsername}
	if err := service.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.created == nil || repo.created.Username != unicodeUsername {
		t.Fatalf("unicode data changed in create path")
	}

	got, err := service.GetByIDForRequester(context.Background(), 7, 1)
	if err != nil {
		t.Fatalf("GetByIDForRequester: %v", err)
	}
	if got == nil || got.Username != unicodeUsername {
		t.Fatalf("unicode data changed in get-by-id path")
	}

	searchResults, err := service.SearchByUsernamePrefixForRequester(context.Background(), unicodePrefix, 1, 10)
	if err != nil {
		t.Fatalf("SearchByUsernamePrefixForRequester: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].Username != unicodeUsername {
		t.Fatalf("unicode data changed in search path")
	}
	if repo.searchForRequesterPrefixArg != unicodePrefix {
		t.Fatalf("unicode prefix changed in forwarded args")
	}

	followers, _, err := service.ListFollowersPageForRequester(context.Background(), 7, 1, nil, 5)
	if err != nil {
		t.Fatalf("ListFollowersPageForRequester: %v", err)
	}
	if len(followers) != 1 || followers[0].Username != unicodeUsername {
		t.Fatalf("unicode data changed in followers page path")
	}
}
