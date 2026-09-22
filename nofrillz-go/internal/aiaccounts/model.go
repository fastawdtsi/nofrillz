package aiaccounts

import "time"

const (
	GenerationStatusIdle    = "idle"
	GenerationStatusRunning = "running"
	GenerationStatusFailed  = "failed"

	PostGenerationStatusGenerated = "generated"
	PostGenerationStatusRejected  = "rejected"
	PostGenerationStatusPosted    = "posted"
	PostGenerationStatusFailed    = "failed"
)

type AIAccount struct {
	ContentMode          string     `json:"content_mode"`
	CheckIntervalSeconds int        `json:"check_interval_seconds"`
	Exclusions           string     `json:"exclusions"`
	SourceURLs           []string   `json:"source_urls"`
	ModelOptions         []string   `json:"model_options"`
	DefaultModelOption   string     `json:"default_model_option"`
	SourceMaxAgeHours    int        `json:"source_max_age_hours"`
	LastCheckedAt        *time.Time `json:"last_checked_at"`
	LastCheckOutcome     string     `json:"last_check_outcome"`

	ID                  uint64     `json:"id,string"`
	UserID              uint64     `json:"user_id,string"`
	Enabled             bool       `json:"enabled"`
	Topic               string     `json:"topic"`
	Description         string     `json:"description"`
	SystemPrompt        string     `json:"system_prompt"`
	StylePrompt         string     `json:"style_prompt"`
	MinPostsPerDay      int        `json:"min_posts_per_day"`
	MaxPostsPerDay      int        `json:"max_posts_per_day"`
	NextGenerateAt      *time.Time `json:"next_generate_at"`
	LastGeneratedAt     *time.Time `json:"last_generated_at"`
	GenerationStatus    string     `json:"generation_status"`
	GenerationStartedAt *time.Time `json:"generation_started_at"`
	GenerationError     string     `json:"generation_error"`
	ClaimToken          string     `json:"-"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type PostGeneration struct {
	ID            uint64    `json:"id,string"`
	AIAccountID   uint64    `json:"ai_account_id,string"`
	PostID        *uint64   `json:"post_id,string"`
	Status        string    `json:"status"`
	Prompt        string    `json:"prompt"`
	CandidateBody string    `json:"candidate_body"`
	FinalBody     string    `json:"final_body"`
	RejectReason  string    `json:"reject_reason"`
	Error         string    `json:"error"`
	Model         string    `json:"model"`
	CreatedAt     time.Time `json:"created_at"`
}
