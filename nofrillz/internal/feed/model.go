package feed

import "time"

type Post struct {
	ID           uint64
	Body         string
	Source       string
	Created      time.Time
	Liked        bool
	IsBookmarked bool
	UserID       uint64
	Username     string
	FirstName    string
	LastName     string
	AccountType  string
}
