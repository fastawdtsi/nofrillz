package posts

import (
	"time"
)

const (
	SourceHuman  = "human"
	SourceAI     = "ai"
	SourceSystem = "system"

	MaxBodyBytes = 65535
)

type PostUser struct {
	UserID      uint64 `json:"user_id"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	AccountType string `json:"account_type"`
	AvatarURL   string `json:"avatar_url"`
}

type Post struct {
	ID           uint64     `json:"id"`
	URL          string     `json:"url"`
	Body         string     `json:"body"`
	Source       string     `json:"source"`
	LikeCount    uint32     `json:"like_count"`
	CommentCount uint32     `json:"comment_count"`
	Liked        bool       `json:"liked"`
	IsBookmarked bool       `json:"is_bookmarked"`
	BookmarkedAt *time.Time `json:"bookmarked_at,omitempty"`
	User         PostUser   `json:"user"`

	Created time.Time  `json:"created"`
	Updated time.Time  `json:"-"`
	Deleted *time.Time `json:"-"`
}
