package testdb

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Explicit opt-in. Every run creates and drops its own database, never the
// application's database. Supply an administrative DSN for a LOCAL MySQL only.
func Open(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("NOFRILLZ_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set NOFRILLZ_TEST_MYSQL_DSN to run isolated MySQL integration tests")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.MultiStatements = true
	root, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("nofrillz_test_%d", time.Now().UnixNano())
	if _, err = root.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4"); err != nil {
		root.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Exec("DROP DATABASE " + name); root.Close() })
	cfg.DBName = name
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := filepath.Glob("../../schema/migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(string(b)); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}
	return db
}
