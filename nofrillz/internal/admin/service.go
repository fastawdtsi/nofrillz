package admin

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/users"
)

var (
	ErrInvalidProfile          = errors.New("invalid profile or persona field length")
	ErrAIAccountNotFound       = errors.New("ai account not found")
	ErrInvalidEmail            = errors.New("invalid email")
	ErrInvalidUsername         = errors.New("invalid username")
	ErrInvalidTopic            = errors.New("invalid topic")
	ErrInvalidDescription      = errors.New("invalid description")
	ErrInvalidSystemPrompt     = errors.New("invalid system prompt")
	ErrInvalidKeywords         = errors.New("invalid keywords")
	ErrInvalidPostsPerDay      = errors.New("invalid posts per day")
	ErrPreviewGenerationFailed = errors.New("preview generation failed")
	ErrPreviewRejected         = errors.New("preview rejected")
	ErrPostGenerationFailed    = errors.New("post generation failed")
	ErrContentGenerationFailed = errors.New("content generation failed")

	ErrInvalidLimit       = errors.New("invalid limit")
	ErrInvalidUserID      = errors.New("invalid user id")
	ErrInvalidPostID      = errors.New("invalid post id")
	ErrInvalidAccountType = errors.New("invalid account type")
	ErrUserNotFound       = errors.New("user not found")
	ErrCannotBlockSystem  = errors.New("cannot block system user")
)

const manualGenerationModel = "admin/manual"

type idGenerator interface {
	MustNext() uint64
}

type moderationRepository interface {
	Stats(ctx context.Context) (*Stats, error)
	ListUsers(ctx context.Context, input ListUsersInput) ([]*UserSummary, error)
	GetUserByID(ctx context.Context, userID uint64) (*UserSummary, error)
	BlockUser(ctx context.Context, userID uint64) error
	ListPosts(ctx context.Context, input ListPostsInput) ([]*PostSummary, error)
	DeletePost(ctx context.Context, postID uint64) (bool, error)
}

type aiAccountsService interface {
	UpdateWithExecutor(context.Context, aiaccounts.UpdateExecutor, *aiaccounts.AIAccount) error
	CreateWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, account *aiaccounts.AIAccount) error
	GetByID(ctx context.Context, id uint64) (*aiaccounts.AIAccount, error)
	ListPostGenerationsByAccountID(ctx context.Context, accountID uint64, limit int) ([]*aiaccounts.PostGeneration, error)
	CreatePostGeneration(ctx context.Context, generation *aiaccounts.PostGeneration) error
	CreatePostGenerationWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, generation *aiaccounts.PostGeneration) error
	MarkGenerationSuccessWithExecutor(ctx context.Context, executor aiaccounts.UpdateExecutor, accountID uint64, lastGeneratedAt time.Time, nextGenerateAt time.Time) error
}

type usersService interface {
	CreateWithExecutor(ctx context.Context, executor users.CreateExecutor, user *users.User) error
	GetByID(ctx context.Context, id uint64) (*users.User, error)
}

type postsService interface {
	CreatePostWithExecutor(ctx context.Context, executor posts.CreateExecutor, input posts.CreatePostInput) (*posts.Post, error)
}

type transaction interface {
	users.CreateExecutor
	posts.CreateExecutor
	aiaccounts.UpdateExecutor
	Commit() error
	Rollback() error
}

type transactionManager interface {
	Begin(ctx context.Context) (transaction, error)
}

type sqlTransactionManager struct {
	db *sql.DB
}

type sqlTransaction struct {
	tx *sql.Tx
}

type Service struct {
	schedule     aiaccounts.Schedule
	transactions transactionManager
	aiAccounts   aiAccountsService
	users        usersService
	posts        postsService
	aiTools      aitools.Tools
	generator    aigenerator.PostGenerator
	idGenerator  idGenerator
	repository   moderationRepository
	now          func() time.Time
}

func NewSQLTransactionManager(db *sql.DB) transactionManager {
	return &sqlTransactionManager{db: db}
}

func (m *sqlTransactionManager) Begin(ctx context.Context) (transaction, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &sqlTransaction{tx: tx}, nil
}

func (t *sqlTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *sqlTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *sqlTransaction) Rollback() error {
	return t.tx.Rollback()
}

func NewService(
	transactions transactionManager,
	aiAccounts aiAccountsService,
	users usersService,
	posts postsService,
	aiTools aitools.Tools,
	generator aigenerator.PostGenerator,
	idGenerator idGenerator,
	repository moderationRepository,
) *Service {
	return &Service{
		transactions: transactions,
		aiAccounts:   aiAccounts,
		users:        users,
		posts:        posts,
		aiTools:      aiTools,
		generator:    generator,
		idGenerator:  idGenerator,
		repository:   repository,
		now:          time.Now,
	}
}

func (s *Service) Stats(ctx context.Context) (*Stats, error) {
	return s.repository.Stats(ctx)
}

func (s *Service) ListUsers(ctx context.Context, input ListUsersInput) ([]*UserSummary, *uint64, error) {
	if input.Limit <= 0 {
		return nil, nil, ErrInvalidLimit
	}
	if !IsSupportedAccountType(input.AccountType) {
		return nil, nil, ErrInvalidAccountType
	}

	results, err := s.repository.ListUsers(ctx, ListUsersInput{
		Query:       input.Query,
		AccountType: input.AccountType,
		Cursor:      input.Cursor,
		Limit:       input.Limit + 1,
	})
	if err != nil {
		return nil, nil, err
	}

	if len(results) <= input.Limit {
		return results, nil, nil
	}

	results = results[:input.Limit]
	last := results[len(results)-1]
	nextCursor := last.ID

	return results, &nextCursor, nil
}

func (s *Service) BlockUser(ctx context.Context, userID uint64) (*UserSummary, error) {
	if userID == 0 {
		return nil, ErrInvalidUserID
	}

	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if user.AccountType == users.AccountTypeSystem {
		return nil, ErrCannotBlockSystem
	}

	if user.BlockedAt == nil {
		if err := s.repository.BlockUser(ctx, userID); err != nil {
			return nil, err
		}
	}

	return s.repository.GetUserByID(ctx, userID)
}

func (s *Service) ListPosts(ctx context.Context, input ListPostsInput) ([]*PostSummary, *uint64, error) {
	if input.Limit <= 0 {
		return nil, nil, ErrInvalidLimit
	}

	results, err := s.repository.ListPosts(ctx, ListPostsInput{
		UserID: input.UserID,
		Cursor: input.Cursor,
		Limit:  input.Limit + 1,
	})
	if err != nil {
		return nil, nil, err
	}

	if len(results) <= input.Limit {
		return results, nil, nil
	}

	results = results[:input.Limit]
	last := results[len(results)-1]
	nextCursor := last.ID

	return results, &nextCursor, nil
}

func (s *Service) DeletePost(ctx context.Context, postID uint64) (bool, error) {
	if postID == 0 {
		return false, ErrInvalidPostID
	}

	return s.repository.DeletePost(ctx, postID)
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (*AccountRecord, error) {
	email := strings.TrimSpace(input.Email)
	username := strings.TrimSpace(input.Username)
	topic := strings.TrimSpace(input.Topic)
	systemPrompt := strings.TrimSpace(input.SystemPrompt)

	if !users.ValidateEmail(email) {
		return nil, ErrInvalidEmail
	}
	if !users.ValidateUsername(username) {
		return nil, ErrInvalidUsername
	}
	if topic == "" {
		return nil, ErrInvalidTopic
	}
	if systemPrompt == "" {
		return nil, ErrInvalidSystemPrompt
	}

	minPostsPerDay := input.MinPostsPerDay
	if minPostsPerDay <= 0 {
		minPostsPerDay = 1
	}
	maxPostsPerDay := input.MaxPostsPerDay
	if maxPostsPerDay <= 0 {
		maxPostsPerDay = 2
	}
	if maxPostsPerDay < minPostsPerDay || maxPostsPerDay > 48 {
		return nil, ErrInvalidPostsPerDay
	}

	if err := validatePersona(input.FirstName, input.LastName, input.About, topic, input.Description, systemPrompt, input.StylePrompt); err != nil {
		return nil, err
	}
	passwordHash, passwordSalt, err := generateRandomCredentials()
	if err != nil {
		return nil, fmt.Errorf("generate ai credentials: %w", err)
	}

	tx, err := s.transactions.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	createdAt := s.now().UTC()
	user := &users.User{
		ID:           s.idGenerator.MustNext(),
		Email:        email,
		Username:     username,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
		About:        strings.TrimSpace(input.About),
		AccountType:  users.AccountTypeAI,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
	}
	if err := s.users.CreateWithExecutor(ctx, tx, user); err != nil {
		return nil, err
	}

	var nextGenerateAt *time.Time
	if input.NextGenerateAt != nil {
		value := input.NextGenerateAt.UTC()
		nextGenerateAt = &value
	} else if input.Enabled {
		value := s.schedule.First(s.now().UTC())
		nextGenerateAt = &value
	}

	account := &aiaccounts.AIAccount{
		ID:               s.idGenerator.MustNext(),
		UserID:           user.ID,
		Enabled:          input.Enabled,
		Topic:            topic,
		Description:      strings.TrimSpace(input.Description),
		SystemPrompt:     systemPrompt,
		StylePrompt:      strings.TrimSpace(input.StylePrompt),
		MinPostsPerDay:   minPostsPerDay,
		MaxPostsPerDay:   maxPostsPerDay,
		NextGenerateAt:   nextGenerateAt,
		GenerationStatus: aiaccounts.GenerationStatusIdle,
		CreatedAt:        createdAt,
		UpdatedAt:        createdAt,
	}
	if err := s.aiAccounts.CreateWithExecutor(ctx, tx, account); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return &AccountRecord{
		Account: account,
		User:    user,
	}, nil
}

func (s *Service) GeneratePostContent(ctx context.Context, input GeneratePostContentInput) (*GeneratePostContentResult, error) {
	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, ErrInvalidDescription
	}

	keywords := normalizeKeywords(input.Keywords)
	if len(keywords) == 0 {
		return nil, ErrInvalidKeywords
	}

	if s.aiTools == nil {
		return nil, ErrContentGenerationFailed
	}

	generated, err := s.aiTools.GeneratePostContent(ctx, aitools.GeneratePostInput{
		Keywords:     keywords,
		Description:  description,
		SystemPrompt: strings.TrimSpace(input.SystemPrompt),
		StylePrompt:  strings.TrimSpace(input.StylePrompt),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContentGenerationFailed, err)
	}

	generated.Body = strings.TrimSpace(generated.Body)
	if generated.Body == "" || len(generated.Body) > posts.MaxBodyBytes {
		return nil, ErrContentGenerationFailed
	}

	return &generated, nil
}

func (s *Service) GetAccount(ctx context.Context, accountID uint64) (*AccountRecord, error) {
	account, err := s.aiAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAIAccountNotFound
	}

	user, err := s.users.GetByID(ctx, account.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrAIAccountNotFound
	}

	return &AccountRecord{
		Account: account,
		User:    user,
	}, nil
}

func (s *Service) ListPostGenerations(ctx context.Context, accountID uint64, limit int) ([]*aiaccounts.PostGeneration, error) {
	account, err := s.aiAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAIAccountNotFound
	}
	if limit <= 0 {
		limit = 20
	}

	return s.aiAccounts.ListPostGenerationsByAccountID(ctx, accountID, limit)
}

func (s *Service) CreatePreview(ctx context.Context, accountID uint64) (*aiaccounts.PostGeneration, error) {
	account, err := s.aiAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAIAccountNotFound
	}

	generated, err := s.generator.GeneratePost(ctx, account)
	if err != nil {
		now := s.now().UTC()
		generation := &aiaccounts.PostGeneration{
			ID:          s.idGenerator.MustNext(),
			AIAccountID: account.ID,
			Status:      aiaccounts.PostGenerationStatusFailed,
			Error:       err.Error(),
			CreatedAt:   now,
		}
		if createErr := s.aiAccounts.CreatePostGeneration(ctx, generation); createErr != nil {
			return nil, createErr
		}
		return generation, ErrPreviewGenerationFailed
	}

	candidateBody := strings.TrimSpace(generated.Body)
	now := s.now().UTC()
	generation := &aiaccounts.PostGeneration{
		ID:            s.idGenerator.MustNext(),
		AIAccountID:   account.ID,
		Status:        aiaccounts.PostGenerationStatusGenerated,
		Prompt:        generated.Prompt,
		CandidateBody: candidateBody,
		Model:         generated.Model,
		CreatedAt:     now,
	}

	if candidateBody == "" || len(candidateBody) > posts.MaxBodyBytes {
		generation.Status = aiaccounts.PostGenerationStatusRejected
		generation.RejectReason = "invalid generated post body"
		if err := s.aiAccounts.CreatePostGeneration(ctx, generation); err != nil {
			return nil, err
		}
		return generation, ErrPreviewRejected
	}

	if err := s.aiAccounts.CreatePostGeneration(ctx, generation); err != nil {
		return nil, err
	}

	return generation, nil
}

func (s *Service) CreatePost(ctx context.Context, accountID uint64, body string) (*CreatePostResult, error) {
	account, err := s.aiAccounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAIAccountNotFound
	}

	manualBody := strings.TrimSpace(body)
	generated := aigenerator.GeneratedPost{}
	model := manualGenerationModel
	prompt := ""
	candidateBody := manualBody

	if manualBody == "" {
		generated, err = s.generator.GeneratePost(ctx, account)
		if err != nil {
			generation := &aiaccounts.PostGeneration{
				ID:          s.idGenerator.MustNext(),
				AIAccountID: account.ID,
				Status:      aiaccounts.PostGenerationStatusFailed,
				Error:       err.Error(),
				CreatedAt:   s.now().UTC(),
			}
			if createErr := s.aiAccounts.CreatePostGeneration(ctx, generation); createErr != nil {
				return nil, createErr
			}
			return &CreatePostResult{Generation: generation}, ErrPostGenerationFailed
		}
		prompt = generated.Prompt
		model = generated.Model
		candidateBody = strings.TrimSpace(generated.Body)
	}

	if candidateBody == "" || len(candidateBody) > posts.MaxBodyBytes {
		generation := &aiaccounts.PostGeneration{
			ID:            s.idGenerator.MustNext(),
			AIAccountID:   account.ID,
			Status:        aiaccounts.PostGenerationStatusRejected,
			Prompt:        prompt,
			CandidateBody: candidateBody,
			RejectReason:  "invalid generated post body",
			Model:         model,
			CreatedAt:     s.now().UTC(),
		}
		if err := s.aiAccounts.CreatePostGeneration(ctx, generation); err != nil {
			return nil, err
		}
		return &CreatePostResult{Generation: generation}, ErrPreviewRejected
	}

	tx, err := s.transactions.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	now := s.now().UTC()
	nextGenerateAt := s.schedule.Next(now, account.MinPostsPerDay, account.MaxPostsPerDay)
	if err := s.aiAccounts.MarkGenerationSuccessWithExecutor(ctx, tx, account.ID, now, nextGenerateAt); err != nil {
		return nil, err
	}
	post, err := s.posts.CreatePostWithExecutor(ctx, tx, posts.CreatePostInput{
		AuthorID: account.UserID,
		Body:     candidateBody,
		Source:   posts.SourceAI,
	})
	if err != nil {
		return nil, err
	}

	postID := post.ID
	generation := &aiaccounts.PostGeneration{
		ID:            s.idGenerator.MustNext(),
		AIAccountID:   account.ID,
		PostID:        &postID,
		Status:        aiaccounts.PostGenerationStatusPosted,
		Prompt:        prompt,
		CandidateBody: candidateBody,
		FinalBody:     post.Body,
		Model:         model,
		CreatedAt:     now,
	}
	if err := s.aiAccounts.CreatePostGenerationWithExecutor(ctx, tx, generation); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return &CreatePostResult{
		Post:       post,
		Generation: generation,
	}, nil
}

func generateRandomCredentials() ([]byte, []byte, error) {
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, nil, err
	}

	password := hex.EncodeToString(passwordBytes)
	return users.GeneratePasswordHashAndSalt(password)
}

func normalizeKeywords(raw []string) []string {
	keywords := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, keyword := range raw {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		keywords = append(keywords, trimmed)
	}

	return keywords
}

func (s *Service) SetSchedule(schedule aiaccounts.Schedule) { s.schedule = schedule }

func validatePersona(first, last, bio, topic, description, personality, style string) error {
	if len(first) > 100 || len(last) > 100 || len(bio) > 1024 || len(topic) > 128 || len(description) > 4000 || len(personality) > 8000 || len(style) > 4000 {
		return ErrInvalidProfile
	}
	return nil
}

func (s *Service) UpdateAccount(ctx context.Context, id uint64, input UpdateAccountInput) (*AccountRecord, error) {
	record, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	a, u := record.Account, record.User
	if u.Blocked != nil || u.Deleted != nil {
		return nil, ErrUserNotFound
	}
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = strings.TrimSpace(*src)
		}
	}
	set(&u.FirstName, input.FirstName)
	set(&u.LastName, input.LastName)
	set(&u.About, input.About)
	set(&a.Topic, input.Topic)
	set(&a.Description, input.Description)
	set(&a.SystemPrompt, input.SystemPrompt)
	set(&a.StylePrompt, input.StylePrompt)
	if input.Enabled != nil {
		a.Enabled = *input.Enabled
	}
	if input.MinPostsPerDay != nil {
		a.MinPostsPerDay = *input.MinPostsPerDay
	}
	if input.MaxPostsPerDay != nil {
		a.MaxPostsPerDay = *input.MaxPostsPerDay
	}
	if a.Topic == "" {
		return nil, ErrInvalidTopic
	}
	if a.SystemPrompt == "" {
		return nil, ErrInvalidSystemPrompt
	}
	if a.MinPostsPerDay < 1 || a.MaxPostsPerDay < a.MinPostsPerDay || a.MaxPostsPerDay > 48 {
		return nil, ErrInvalidPostsPerDay
	}
	if err := validatePersona(u.FirstName, u.LastName, u.About, a.Topic, a.Description, a.SystemPrompt, a.StylePrompt); err != nil {
		return nil, err
	}
	a.NextGenerateAt = nil
	if a.Enabled {
		next := s.schedule.First(s.now().UTC())
		a.NextGenerateAt = &next
	}
	tx, err := s.transactions.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Cancel any in-flight claim when configuration changes. Its eventual result
	// will be discarded by the worker's conditional finalization.
	if err := s.aiAccounts.UpdateWithExecutor(ctx, tx, a); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET first_name=?,last_name=?,about=? WHERE id=?", u.FirstName, u.LastName, u.About, u.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, id)
}
