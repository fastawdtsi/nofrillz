package admin

import (
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
)

type Stats struct {
	UserCount             uint64 `json:"user_count"`
	HumanUserCount        uint64 `json:"human_user_count"`
	AIUserCount           uint64 `json:"ai_user_count"`
	BlockedUserCount      uint64 `json:"blocked_user_count"`
	PostCount             uint64 `json:"post_count"`
	AIPostCount           uint64 `json:"ai_post_count"`
	EnabledAIAccountCount uint64 `json:"enabled_ai_account_count"`
}

type UserSummary struct {
	AIEnabled          bool       `json:"ai_enabled"`
	AINextPostAt       *time.Time `json:"ai_next_post_at,omitempty"`
	AILastPostAt       *time.Time `json:"ai_last_post_at,omitempty"`
	AIGenerationStatus string     `json:"ai_generation_status,omitempty"`
	AIGenerationError  string     `json:"ai_generation_error,omitempty"`
	AIMinPostsPerDay   int        `json:"ai_min_posts_per_day"`
	AIMaxPostsPerDay   int        `json:"ai_max_posts_per_day"`

	ID          uint64     `json:"id,string"`
	Email       string     `json:"email"`
	Username    string     `json:"username"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	About       string     `json:"about"`
	AccountType string     `json:"account_type"`
	AvatarURL   string     `json:"avatar_url"`
	AIAccountID *uint64    `json:"ai_account_id,omitempty,string"`
	BlockedAt   *time.Time `json:"blocked_at,omitempty"`
	PostCount   uint64     `json:"post_count"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PostUserSummary struct {
	UserID      uint64 `json:"user_id,string"`
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	AccountType string `json:"account_type"`
	AvatarURL   string `json:"avatar_url"`
}

type PostSummary struct {
	ID        uint64          `json:"id,string"`
	URL       string          `json:"url"`
	Body      string          `json:"body"`
	Source    string          `json:"source"`
	User      PostUserSummary `json:"user"`
	CreatedAt time.Time       `json:"created_at"`
}

type ListUsersInput struct {
	Query       string
	AccountType string
	Cursor      *uint64
	Limit       int
}

type ListPostsInput struct {
	UserID *uint64
	Cursor *uint64
	Limit  int
}

type AccountRecord struct {
	Account *aiaccounts.AIAccount `json:"account"`
	User    *users.User           `json:"user"`
}

type CreateAccountInput struct {
	Email          string
	Username       string
	FirstName      string
	LastName       string
	About          string
	Enabled        bool
	Topic          string
	Description    string
	SystemPrompt   string
	StylePrompt    string
	MinPostsPerDay int
	MaxPostsPerDay int
	NextGenerateAt *time.Time
}

type CreatePostResult struct {
	Post       *posts.Post                `json:"post"`
	Generation *aiaccounts.PostGeneration `json:"generation"`
}

type GeneratePostContentInput struct {
	Keywords     []string
	Description  string
	SystemPrompt string
	StylePrompt  string
}

type GeneratePostContentResult = aitools.GeneratedPost

// Pointer fields preserve omitted values for PATCH, including enabled=false.
type UpdateAccountInput struct {
	FirstName      *string `json:"first_name"`
	LastName       *string `json:"last_name"`
	About          *string `json:"about"`
	Enabled        *bool   `json:"enabled"`
	Topic          *string `json:"topic"`
	Description    *string `json:"description"`
	SystemPrompt   *string `json:"system_prompt"`
	StylePrompt    *string `json:"style_prompt"`
	MinPostsPerDay *int    `json:"min_posts_per_day"`
	MaxPostsPerDay *int    `json:"max_posts_per_day"`
}
