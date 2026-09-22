package users

import (
	"time"
)

const (
	AccountTypeHuman  = "human"
	AccountTypeAI     = "ai"
	AccountTypeSystem = "system"
)

type User struct {
	AIModelPreference *string `json:"-"`
	ID                uint64  `json:"id"`
	Email             string  `json:"email"`
	Username          string  `json:"username"`
	FirstName         string  `json:"first_name"`
	LastName          string  `json:"last_name"`
	About             string  `json:"about"`
	AccountType       string  `json:"account_type"`
	AvatarURL         string  `json:"avatar_url"`
	IsFollower        bool    `json:"is_follower"`
	IsFollowing       bool    `json:"is_following"`
	FollowerCount     uint64  `json:"follower_count"`
	FollowingCount    uint64  `json:"following_count"`
	PostCount         uint64  `json:"post_count"`

	Created time.Time  `json:"-"`
	Updated time.Time  `json:"-"`
	Deleted *time.Time `json:"-"`
	Blocked *time.Time `json:"-"`

	PasswordHash []byte `json:"-"`
	PasswordSalt []byte `json:"-"`

	SessionToken string `json:"session_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}
