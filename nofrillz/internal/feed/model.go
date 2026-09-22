package feed

import "time"

type Post struct {
	SortID        uint64
	ContentItemID *uint64
	ModelOption   string
	Provider      string
	Model         string

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
