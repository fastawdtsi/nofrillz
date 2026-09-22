package notifications

import "time"

type Settings struct {
	UserID       uint64 `json:"-"`
	Enabled      bool   `json:"enabled"`
	NewFollowers bool   `json:"new_followers"`
	NewLikes     bool   `json:"new_likes"`
	Replies      bool   `json:"replies"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type UpdateSettingsInput struct {
	Enabled      *bool
	NewFollowers *bool
	NewLikes     *bool
	Replies      *bool
}
