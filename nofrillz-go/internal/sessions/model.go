package sessions

import "time"

type Session struct {
	ID      uint64
	UserID  uint64
	Created time.Time
	Updated time.Time
	Revoked *time.Time
}

type RefreshToken struct {
	TokenHash string
	SessionID uint64
	UserID    uint64
	Created   time.Time
	ExpiresAt time.Time
	Revoked   *time.Time
}
