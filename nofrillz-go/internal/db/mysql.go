package db

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"nofrillz/internal/config"
)

type MySQL struct {
	DB *sql.DB
}

func NewMySQL(config *config.MySQLConfig) (*MySQL, error) {
	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &MySQL{DB: db}, nil
}

func (m *MySQL) Close() error {
	if m == nil || m.DB == nil {
		return nil
	}

	return m.DB.Close()
}
