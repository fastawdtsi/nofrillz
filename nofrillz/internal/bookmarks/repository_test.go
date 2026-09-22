package bookmarks

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	bookmarkRepositoryDriverOnce sync.Once
	bookmarkRepositoryDriverDBs  sync.Map
	bookmarkRepositoryDriverSeq  uint64
)

type bookmarkRepositoryExecExpectation struct {
	queryContains string
	args          []any
	rowsAffected  int64
	err           error
}

type bookmarkRepositoryQueryExpectation struct {
	queryContains string
	args          []any
	columns       []string
	rows          [][]driver.Value
	err           error
}

type bookmarkRepositoryTestDB struct {
	t *testing.T

	mu      sync.Mutex
	execs   []bookmarkRepositoryExecExpectation
	queries []bookmarkRepositoryQueryExpectation
}

type bookmarkRepositoryDriver struct{}

type bookmarkRepositoryConn struct {
	stub *bookmarkRepositoryTestDB
}

type bookmarkRepositoryRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func newBookmarkRepositoryTestDB(t *testing.T) (*sql.DB, *bookmarkRepositoryTestDB) {
	t.Helper()

	bookmarkRepositoryDriverOnce.Do(func() {
		sql.Register("bookmark_repository_test_driver", &bookmarkRepositoryDriver{})
	})

	dsn := fmt.Sprintf("bookmark-repository-%d", atomic.AddUint64(&bookmarkRepositoryDriverSeq, 1))
	stub := &bookmarkRepositoryTestDB{t: t}
	bookmarkRepositoryDriverDBs.Store(dsn, stub)

	db, err := sql.Open("bookmark_repository_test_driver", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close: %v", err)
		}
		bookmarkRepositoryDriverDBs.Delete(dsn)
		stub.assertExpectationsMet()
	})

	return db, stub
}

func (s *bookmarkRepositoryTestDB) assertExpectationsMet() {
	s.t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.execs) > 0 {
		s.t.Fatalf("unmet exec expectations: %d", len(s.execs))
	}
	if len(s.queries) > 0 {
		s.t.Fatalf("unmet query expectations: %d", len(s.queries))
	}
}

func (s *bookmarkRepositoryTestDB) handleExec(query string, args []driver.NamedValue) (driver.Result, error) {
	s.t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.execs) == 0 {
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}

	expectation := s.execs[0]
	s.execs = s.execs[1:]

	if expectation.queryContains != "" && !strings.Contains(query, expectation.queryContains) {
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
	if err := compareNamedValues(expectation.args, args); err != nil {
		return nil, fmt.Errorf("unexpected exec args: %w", err)
	}
	if expectation.err != nil {
		return nil, expectation.err
	}

	return driver.RowsAffected(expectation.rowsAffected), nil
}

func (s *bookmarkRepositoryTestDB) handleQuery(query string, args []driver.NamedValue) (driver.Rows, error) {
	s.t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.queries) == 0 {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}

	expectation := s.queries[0]
	s.queries = s.queries[1:]

	if expectation.queryContains != "" && !strings.Contains(query, expectation.queryContains) {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	if err := compareNamedValues(expectation.args, args); err != nil {
		return nil, fmt.Errorf("unexpected query args: %w", err)
	}
	if expectation.err != nil {
		return nil, expectation.err
	}

	return &bookmarkRepositoryRows{
		columns: expectation.columns,
		rows:    expectation.rows,
	}, nil
}

func (d *bookmarkRepositoryDriver) Open(name string) (driver.Conn, error) {
	stub, ok := bookmarkRepositoryDriverDBs.Load(name)
	if !ok {
		return nil, errors.New("missing bookmark repository test db")
	}

	return &bookmarkRepositoryConn{stub: stub.(*bookmarkRepositoryTestDB)}, nil
}

func (c *bookmarkRepositoryConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *bookmarkRepositoryConn) Close() error {
	return nil
}

func (c *bookmarkRepositoryConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions not supported")
}

func (c *bookmarkRepositoryConn) CheckNamedValue(*driver.NamedValue) error {
	return nil
}

func (c *bookmarkRepositoryConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.stub.handleExec(query, args)
}

func (c *bookmarkRepositoryConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.stub.handleQuery(query, args)
}

func (r *bookmarkRepositoryRows) Columns() []string {
	return r.columns
}

func (r *bookmarkRepositoryRows) Close() error {
	return nil
}

func (r *bookmarkRepositoryRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}

	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func compareNamedValues(want []any, got []driver.NamedValue) error {
	if len(want) != len(got) {
		return fmt.Errorf("want %d args, got %d", len(want), len(got))
	}

	for i := range want {
		if !equalBookmarkRepositoryValue(want[i], got[i].Value) {
			return fmt.Errorf("arg %d mismatch: want %#v, got %#v", i, want[i], got[i].Value)
		}
	}

	return nil
}

func equalBookmarkRepositoryValue(want any, got any) bool {
	wantTime, wantIsTime := want.(time.Time)
	gotTime, gotIsTime := got.(time.Time)
	if wantIsTime || gotIsTime {
		return wantIsTime && gotIsTime && wantTime.Equal(gotTime)
	}

	wantNumber, wantIsNumber := numericBookmarkRepositoryValue(want)
	gotNumber, gotIsNumber := numericBookmarkRepositoryValue(got)
	if wantIsNumber || gotIsNumber {
		return wantIsNumber && gotIsNumber && wantNumber == gotNumber
	}

	return reflect.DeepEqual(want, got)
}

func numericBookmarkRepositoryValue(value any) (uint64, bool) {
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue := reflectValue.Int()
		if intValue < 0 {
			return 0, false
		}
		return uint64(intValue), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflectValue.Uint(), true
	default:
		return 0, false
	}
}

func TestRepositoryBookmarkPost(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	stub.execs = []bookmarkRepositoryExecExpectation{
		{
			queryContains: "insert ignore into post_bookmarks",
			args:          []any{uint64(44), uint64(55)},
			rowsAffected:  1,
		},
	}

	repository := NewRepository(db)
	created, err := repository.BookmarkPost(context.Background(), 55, 44)
	if err != nil {
		t.Fatalf("BookmarkPost: %v", err)
	}
	if !created {
		t.Fatalf("expected bookmark creation to be reported")
	}
}

func TestRepositoryBookmarkPostDuplicate(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	stub.execs = []bookmarkRepositoryExecExpectation{
		{
			queryContains: "insert ignore into post_bookmarks",
			args:          []any{uint64(44), uint64(55)},
			rowsAffected:  0,
		},
	}

	repository := NewRepository(db)
	created, err := repository.BookmarkPost(context.Background(), 55, 44)
	if err != nil {
		t.Fatalf("BookmarkPost: %v", err)
	}
	if created {
		t.Fatalf("expected duplicate bookmark to report created=false")
	}
}

func TestRepositoryUnbookmarkPost(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	stub.execs = []bookmarkRepositoryExecExpectation{
		{
			queryContains: "delete from post_bookmarks",
			args:          []any{uint64(44), uint64(55)},
			rowsAffected:  1,
		},
	}

	repository := NewRepository(db)
	removed, err := repository.UnbookmarkPost(context.Background(), 55, 44)
	if err != nil {
		t.Fatalf("UnbookmarkPost: %v", err)
	}
	if !removed {
		t.Fatalf("expected bookmark removal to be reported")
	}
}

func TestRepositoryUnbookmarkPostMissing(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	stub.execs = []bookmarkRepositoryExecExpectation{
		{
			queryContains: "delete from post_bookmarks",
			args:          []any{uint64(44), uint64(55)},
			rowsAffected:  0,
		},
	}

	repository := NewRepository(db)
	removed, err := repository.UnbookmarkPost(context.Background(), 55, 44)
	if err != nil {
		t.Fatalf("UnbookmarkPost: %v", err)
	}
	if removed {
		t.Fatalf("expected missing bookmark to report removed=false")
	}
}

func TestRepositoryIsPostBookmarked(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	stub.queries = []bookmarkRepositoryQueryExpectation{
		{
			queryContains: "select exists(",
			args:          []any{uint64(44), uint64(55)},
			columns:       []string{"bookmarked"},
			rows:          [][]driver.Value{{true}},
		},
	}

	repository := NewRepository(db)
	bookmarked, err := repository.IsPostBookmarked(context.Background(), 55, 44)
	if err != nil {
		t.Fatalf("IsPostBookmarked: %v", err)
	}
	if !bookmarked {
		t.Fatalf("expected bookmarked=true")
	}
}

func TestRepositoryListBookmarkedPosts(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	now := time.Date(2026, 7, 11, 18, 0, 0, 0, time.UTC)
	later := now.Add(-1 * time.Minute)
	stub.queries = []bookmarkRepositoryQueryExpectation{
		{
			queryContains: "from post_bookmarks pb",
			args:          []any{uint64(44), uint64(44), 3},
			columns: []string{
				"id", "user_id", "body", "source", "created", "updated", "deleted",
				"username", "first_name", "last_name", "account_type", "liked", "is_bookmarked", "bookmarked_at",
			},
			rows: [][]driver.Value{
				{int64(55), int64(44), "saved first", "human", now, now, nil, "alice", "Alice", "Anderson", "human", true, true, now},
				{int64(54), int64(44), "saved second", "ai", later, later, nil, "alice", "Alice", "Anderson", "human", false, true, later},
			},
		},
	}

	repository := NewRepository(db)
	results, err := repository.ListBookmarkedPosts(context.Background(), 44, nil, 3)
	if err != nil {
		t.Fatalf("ListBookmarkedPosts: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != 55 || results[1].ID != 54 {
		t.Fatalf("unexpected ordering: %d, %d", results[0].ID, results[1].ID)
	}
	if !results[0].IsBookmarked || !results[1].IsBookmarked {
		t.Fatalf("expected all listed posts to be bookmarked")
	}
	if results[0].BookmarkedAt == nil || !results[0].BookmarkedAt.Equal(now) {
		t.Fatalf("expected bookmarked_at to be populated for first result")
	}
	if results[1].BookmarkedAt == nil || !results[1].BookmarkedAt.Equal(later) {
		t.Fatalf("expected bookmarked_at to be populated for second result")
	}
}

func TestRepositoryListBookmarkedPostsWithCursor(t *testing.T) {
	db, stub := newBookmarkRepositoryTestDB(t)
	now := time.Date(2026, 7, 11, 18, 0, 0, 0, time.UTC)
	cursor := &Cursor{Created: now, PostID: 55}
	stub.queries = []bookmarkRepositoryQueryExpectation{
		{
			queryContains: "pb.created < ?",
			args:          []any{uint64(44), uint64(44), now, now, uint64(55), 2},
			columns: []string{
				"id", "user_id", "body", "source", "created", "updated", "deleted",
				"username", "first_name", "last_name", "account_type", "liked", "is_bookmarked", "bookmarked_at",
			},
			rows: [][]driver.Value{
				{int64(54), int64(44), "saved second", "human", now.Add(-time.Minute), now.Add(-time.Minute), nil, "alice", "Alice", "Anderson", "human", false, true, now.Add(-time.Minute)},
			},
		},
	}

	repository := NewRepository(db)
	results, err := repository.ListBookmarkedPosts(context.Background(), 44, cursor, 2)
	if err != nil {
		t.Fatalf("ListBookmarkedPosts: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != 54 {
		t.Fatalf("expected cursor query to return post 54, got %d", results[0].ID)
	}
}
