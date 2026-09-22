package aicontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/aitools"
	"nofrillz/internal/posts"
	"nofrillz/internal/research"
)

type IDs interface{ MustNext() uint64 }
type Item struct {
	ID          uint64            `json:"id,string"`
	AccountID   uint64            `json:"ai_account_id,string"`
	Key         string            `json:"dedup_key"`
	Title       string            `json:"title"`
	Context     string            `json:"context"`
	Sources     []research.Source `json:"sources"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	PublishedAt *time.Time        `json:"published_at"`
	Variants    []Variant         `json:"variants"`
}
type Variant struct {
	OptionID string  `json:"option_id"`
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	PostID   *uint64 `json:"post_id,omitempty,string"`
	Body     string  `json:"body,omitempty"`
	Status   string  `json:"status"`
	Error    string  `json:"error,omitempty"`
}
type Service struct {
	DB       *sql.DB
	Registry *aitools.Registry
	Research research.Researcher
	Posts    *posts.Service
	Accounts *aiaccounts.Repository
	IDs      IDs
	Schedule aiaccounts.Schedule
	Logger   *zerolog.Logger
	Now      func() time.Time
}

func New(db *sql.DB, registry *aitools.Registry, sources research.Researcher, postService *posts.Service, ids IDs, schedule aiaccounts.Schedule, logger *zerolog.Logger) *Service {
	return &Service{db, registry, sources, postService, aiaccounts.NewRepository(db), ids, schedule, logger, time.Now}
}
func (s *Service) log(a *aiaccounts.AIAccount, msg string, fields map[string]any) {
	if s.Logger != nil {
		s.Logger.Info().Uint64("ai_account_id", a.ID).Uint64("user_id", a.UserID).Fields(fields).Msg(msg)
	}
}

// Every mutation checks and locks claim ownership. Pausing/editing or recovering
// an expired lease fences the old worker before it can create an item or post.
func (s *Service) withClaim(ctx context.Context, a *aiaccounts.AIAccount, work func(*sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id uint64
	err = tx.QueryRowContext(ctx, `SELECT a.id FROM ai_accounts a JOIN users u ON u.id=a.user_id WHERE a.id=? AND a.claim_token=? AND a.enabled=TRUE AND a.generation_status='running' AND u.deleted IS NULL AND u.blocked IS NULL FOR UPDATE`, a.ID, a.ClaimToken).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return aiaccounts.ErrClaimLost
	}
	if err != nil {
		return err
	}
	if err = work(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) Process(ctx context.Context, a *aiaccounts.AIAccount) error {
	if a == nil || !a.Enabled || a.ClaimToken == "" {
		return aiaccounts.ErrClaimLost
	}
	s.log(a, "content account check started", map[string]any{"mode": a.ContentMode, "beat": a.Topic})
	item, err := s.pending(ctx, a.ID)
	if err != nil {
		return s.fail(ctx, a, err)
	}
	if item == nil {
		item, err = s.prepare(ctx, a)
		if err != nil {
			return s.fail(ctx, a, err)
		}
		if item == nil {
			return s.finish(ctx, a, nil, false, "no_content", "")
		}
	}
	successes := 0
	failures := 0
	unavailableReviewOptions := map[string]bool{}
	for _, optionID := range a.ModelOptions {
		existing, err := s.variant(ctx, item.ID, optionID)
		if err != nil {
			return s.fail(ctx, a, err)
		}
		if existing != nil {
			if existing.Status == "duplicate" && successes == 0 {
				return s.finish(ctx, a, item, false, "duplicate", "")
			}
			if existing.Status == "skipped" && a.ContentMode == "research" && successes == 0 {
				return s.finish(ctx, a, item, false, "not_significant", "")
			}
			if existing.PostID != nil {
				successes++
			} else {
				if existing.Status == "attempting" {
					if err = s.saveFailure(ctx, a, item, aitools.Option{ID: optionID, Provider: existing.Provider, Model: existing.Model}, "attempt interrupted; not automatically retried"); err != nil {
						return err
					}
				}
				failures++
			}
			continue
		}
		option, ok := s.Registry.Get(optionID)
		if !ok {
			option = aitools.Option{ID: optionID, Provider: "unconfigured", Model: "unconfigured"}
		}
		if !ok || !option.Available {
			if err = s.saveFailure(ctx, a, item, option, "provider is not configured"); err != nil {
				return err
			}
			failures++
			continue
		}
		// Persist an attempt before calling an external provider. A crash leaves a
		// failed/unknown attempt on recovery rather than spending again automatically.
		if err = s.withClaim(ctx, a, func(tx *sql.Tx) error {
			_, e := tx.ExecContext(ctx, `INSERT INTO ai_content_variants(content_item_id,option_id,provider,model,status,error) VALUES(?,?,?,?,'attempting','attempt interrupted before persistence')`, item.ID, option.ID, option.Provider, option.Model)
			return e
		}); err != nil {
			return err
		}
		recent, err := s.recent(ctx, a.UserID, item.ID)
		if err != nil {
			return s.fail(ctx, a, err)
		}
		sources, _ := json.Marshal(item.Sources)
		input := aitools.GeneratePostInput{ContentMode: a.ContentMode, Description: a.Description, Keywords: []string{a.Topic}, SystemPrompt: a.SystemPrompt, StylePrompt: a.StylePrompt, Exclusions: a.Exclusions, Context: item.Context, Sources: string(sources), SharedDraft: a.ContentMode == "generative" && item.Context != "", RecentPosts: recent, LengthHint: "Use the length the mission needs: usually one to three sentences; at most 1200 characters."}
		s.log(a, "generating content variant", map[string]any{"content_item_id": item.ID, "option_id": option.ID, "provider": option.Provider, "model": option.Model})
		generated, genErr := option.Tools.GeneratePostContent(ctx, input)
		if genErr != nil {
			unavailableReviewOptions[option.ID] = true
		}
		if genErr == nil && strings.TrimSpace(generated.Body) == "__NO_POST__" {
			if a.ContentMode == "research" && successes == 0 {
				return s.skipResearch(ctx, a, item, option.ID)
			}
			genErr = fmt.Errorf("model declined this content item")
		}
		if genErr == nil {
			// Format validation is separate from novelty. Similar wording may carry
			// genuinely new facts; compare meaning before rejecting those updates.
			genErr = aigenerator.ValidateCandidate(strings.TrimSpace(generated.Body), nil)
		}
		if genErr == nil && (a.ContentMode == "research" || item.Context != "") {
			// A shared factual editor checks each presentation against the original
			// evidence. It may reject a variant, but never rewrites its viewpoint.
			reviewInput := aitools.GeneratePostInput{ContentMode: a.ContentMode, ReviewBody: generated.Body, Context: item.Context, Sources: string(sources), Description: a.Description, Exclusions: a.Exclusions}
			review, reviewer, e := s.review(ctx, a, option, unavailableReviewOptions, reviewInput)
			if e == nil && a.ContentMode == "research" && successes == 0 && strings.TrimSpace(review.Body) == "__NO_POST__" {
				s.log(a, "research item excluded by editorial review", map[string]any{"content_item_id": item.ID, "review_option": reviewer.ID})
				return s.skipResearch(ctx, a, item, option.ID)
			}
			if e != nil {
				genErr = fmt.Errorf("content evidence review failed: %w", e)
			} else if strings.TrimSpace(review.Body) != "__APPROVED__" {
				if a.ContentMode == "research" {
					genErr = fmt.Errorf("research draft contains claims not supported by the retrieved source")
				} else {
					genErr = fmt.Errorf("variant does not preserve the accepted content seed")
				}
			}
			s.log(a, "content evidence reviewed", map[string]any{"mode": a.ContentMode, "content_item_id": item.ID, "option_id": option.ID, "review_option": reviewer.ID, "review_model": review.Model, "approved": genErr == nil})
		}
		if genErr == nil {
			var duplicate *aitools.PriorContent
			duplicate, genErr = s.checkNovelty(ctx, a, item, option, unavailableReviewOptions, strings.TrimSpace(generated.Body))
			if genErr == nil && duplicate != nil {
				if err = s.saveDuplicate(ctx, a, item, option.ID, duplicate); err != nil {
					return err
				}
				s.log(a, "duplicate content suppressed", map[string]any{"content_item_id": item.ID, "option_id": option.ID, "previous_post_id": duplicate.PostID, "previous_content_item_id": duplicate.ItemID})
				if successes == 0 {
					// Do not spend on more presentations of an already-covered item.
					return s.finish(ctx, a, item, false, "duplicate", "")
				}
				failures++
				continue
			}
		}
		if genErr != nil {
			if err = s.saveFailure(ctx, a, item, option, genErr.Error()); err != nil {
				return err
			}
			failures++
			s.log(a, "content variant failed", map[string]any{"content_item_id": item.ID, "option_id": option.ID, "error": safeMessage(genErr.Error())})
			continue
		}
		var publishedPostID uint64
		err = s.withClaim(ctx, a, func(tx *sql.Tx) error {
			post, e := s.Posts.CreatePostWithExecutor(ctx, tx, posts.CreatePostInput{AuthorID: a.UserID, Body: strings.TrimSpace(generated.Body), Source: posts.SourceAI})
			if e != nil {
				return e
			}
			_, e = tx.ExecContext(ctx, `UPDATE ai_content_variants SET post_id=?,model=?,status='published',error=NULL WHERE content_item_id=? AND option_id=? AND post_id IS NULL`, post.ID, generated.Model, item.ID, option.ID)
			if e != nil {
				return e
			}
			if a.ContentMode == "generative" && item.Context == "" {
				item.Context = post.Body
			}
			_, e = tx.ExecContext(ctx, `UPDATE ai_content_items SET context=?,published_at=COALESCE(published_at,?) WHERE id=?`, item.Context, s.Now().UTC(), item.ID)
			if e != nil {
				return e
			}
			generation := &aiaccounts.PostGeneration{ID: s.IDs.MustNext(), AIAccountID: a.ID, PostID: &post.ID, Status: aiaccounts.PostGenerationStatusPosted, Prompt: generated.Prompt, CandidateBody: generated.Body, FinalBody: post.Body, Model: generated.Model}
			if e = s.Accounts.CreatePostGenerationWithExecutor(ctx, tx, generation); e != nil {
				return e
			}
			publishedPostID = post.ID
			return nil
		})
		if err != nil {
			return s.fail(ctx, a, err)
		}
		s.log(a, "content variant created", map[string]any{"content_item_id": item.ID, "option_id": option.ID, "post_id": publishedPostID})
		successes++
	}
	if successes == 0 {
		return s.finish(ctx, a, item, false, "failed", "all configured variants failed or are unavailable")
	}
	outcome := "published"
	if failures > 0 {
		outcome = "partially_published"
	}
	return s.finish(ctx, a, item, true, outcome, "")
}
func (s *Service) skipResearch(ctx context.Context, a *aiaccounts.AIAccount, item *Item, optionID string) error {
	if err := s.withClaim(ctx, a, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE ai_content_variants SET status='skipped',error=NULL WHERE content_item_id=? AND option_id=?`, item.ID, optionID)
		return err
	}); err != nil {
		return err
	}
	return s.finish(ctx, a, item, false, "not_significant", "")
}

func safeMessage(s string) string {
	r := []rune(s)
	if len(r) > 500 {
		r = r[:500]
	}
	return string(r)
}
func (s *Service) saveFailure(ctx context.Context, a *aiaccounts.AIAccount, item *Item, option aitools.Option, message string) error {
	return s.withClaim(ctx, a, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO ai_content_variants(content_item_id,option_id,provider,model,status,error) VALUES(?,?,?,?,'failed',?) ON DUPLICATE KEY UPDATE status=IF(post_id IS NULL,'failed',status),error=IF(post_id IS NULL,VALUES(error),error)`, item.ID, option.ID, option.Provider, option.Model, safeMessage(message))
		return err
	})
}
func (s *Service) prepare(ctx context.Context, a *aiaccounts.AIAccount) (*Item, error) {
	item := &Item{ID: s.IDs.MustNext(), AccountID: a.ID, Title: a.Topic, Status: "processing", Sources: []research.Source{}, CreatedAt: s.Now().UTC()}
	keys := []string{}
	if a.ContentMode == "research" {
		s.log(a, "research started", nil)
		candidates, err := s.Research.Discover(ctx, research.Request{URLs: a.SourceURLs, MaxAge: time.Duration(a.SourceMaxAgeHours) * time.Hour, Now: s.Now().UTC()})
		if err != nil {
			return nil, err
		}
		previously := 0
		for _, candidate := range candidates {
			titleKey := research.Fingerprint("title:" + strings.ToLower(strings.Join(strings.Fields(candidate.Title), " ")))
			var seen int
			if err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ai_processed_sources WHERE ai_account_id=? AND fingerprint IN (?,?)`, a.ID, candidate.Key, titleKey).Scan(&seen); err != nil {
				return nil, err
			}
			if seen > 0 {
				previously++
				continue
			}
			item.Key = candidate.Key
			item.Title = candidate.Title
			item.Context = candidate.Context
			item.Sources = candidate.Sources
			keys = []string{candidate.Key, titleKey}
			break
		}
		s.log(a, "research candidates evaluated", map[string]any{"discovered": len(candidates), "previously_processed": previously, "selected": len(keys) > 0})
		if len(keys) == 0 {
			return nil, nil
		}
	} else if a.ContentMode == "generative" {
		item.Key = research.Fingerprint("generation:" + a.ClaimToken)
	} else {
		return nil, fmt.Errorf("unsupported content mode")
	}
	sourceJSON, _ := json.Marshal(item.Sources)
	err := s.withClaim(ctx, a, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO ai_content_items(id,ai_account_id,dedup_key,title,context,sources,status) VALUES(?,?,?,?,?,?,'processing')`, item.ID, a.ID, item.Key, item.Title, item.Context, string(sourceJSON))
		if err != nil {
			return err
		}
		for _, key := range keys {
			if _, err = tx.ExecContext(ctx, `INSERT INTO ai_processed_sources(ai_account_id,fingerprint,content_item_id,outcome) VALUES(?,?,?,'evaluated')`, a.ID, key, item.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.log(a, "logical content item created", map[string]any{"content_item_id": item.ID})
	return item, nil
}
func (s *Service) fail(ctx context.Context, a *aiaccounts.AIAccount, err error) error {
	if errors.Is(err, aiaccounts.ErrClaimLost) {
		return err
	}
	if e := s.finish(ctx, a, nil, false, "failed", safeMessage(err.Error())); e != nil {
		return e
	}
	return err
}
func (s *Service) finish(ctx context.Context, a *aiaccounts.AIAccount, item *Item, published bool, outcome, message string) error {
	now := s.Now().UTC()
	next := s.Schedule.NextCheck(now, a.CheckIntervalSeconds)
	if outcome == "failed" {
		base := 15 * time.Minute * time.Duration(1<<min(a.ConsecutiveFailures, 4))
		backoff := now.Add(base + time.Duration(now.UnixNano()%int64(base/2+1)))
		if backoff.After(next) {
			next = backoff
		}
	}
	err := s.withClaim(ctx, a, func(tx *sql.Tx) error {
		var last any
		if published {
			last = now
		}
		_, err := tx.ExecContext(ctx, `UPDATE ai_accounts SET generation_status='idle',claim_token=NULL,generation_started_at=NULL,last_checked_at=?,last_check_outcome=?,last_generated_at=COALESCE(?,last_generated_at),next_generate_at=?,generation_error=?,consecutive_failures=IF(?,consecutive_failures+1,0) WHERE id=?`, now, outcome, last, next, message, outcome == "failed", a.ID)
		if err != nil {
			return err
		}
		if item != nil {
			state := "complete"
			if outcome == "not_significant" || outcome == "duplicate" {
				state = "skipped"
			}
			if outcome == "failed" {
				state = "failed"
			}
			_, err = tx.ExecContext(ctx, `UPDATE ai_content_items SET status=?,completed_at=? WHERE id=?`, state, now, item.ID)
		}
		return err
	})
	if err == nil {
		s.log(a, "content check finished", map[string]any{"outcome": outcome, "next_check_at": next, "error": message})
	}
	return err
}
func (s *Service) recent(ctx context.Context, userID uint64, itemID uint64) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT body FROM (
SELECT p.id,p.body,ROW_NUMBER() OVER (PARTITION BY COALESCE(v.content_item_id,p.id) ORDER BY p.id DESC) AS item_rank
FROM posts p LEFT JOIN ai_content_variants v ON v.post_id=p.id
WHERE p.user_id=? AND (v.content_item_id IS NULL OR v.content_item_id<>?)
) history WHERE item_rank=1 ORDER BY id DESC LIMIT 15`, userID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var body string
		if err = rows.Scan(&body); err != nil {
			return nil, err
		}
		out = append(out, body)
	}
	return out, rows.Err()
}
func scanItem(scanner interface{ Scan(...any) error }) (*Item, error) {
	item := &Item{}
	var sources []byte
	if err := scanner.Scan(&item.ID, &item.AccountID, &item.Key, &item.Title, &item.Context, &sources, &item.Status, &item.CreatedAt, &item.PublishedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(sources, &item.Sources); err != nil {
		return nil, err
	}
	item.Variants = []Variant{}
	return item, nil
}

const itemColumns = `id,ai_account_id,dedup_key,title,context,sources,status,created_at,published_at`

func (s *Service) pending(ctx context.Context, accountID uint64) (*Item, error) {
	item, err := scanItem(s.DB.QueryRowContext(ctx, `SELECT `+itemColumns+` FROM ai_content_items WHERE ai_account_id=? AND status='processing' ORDER BY id LIMIT 1`, accountID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}
func (s *Service) variant(ctx context.Context, itemID uint64, option string) (*Variant, error) {
	v := &Variant{}
	err := s.DB.QueryRowContext(ctx, `SELECT option_id,provider,model,post_id,status,COALESCE(error,'') FROM ai_content_variants WHERE content_item_id=? AND option_id=?`, itemID, option).Scan(&v.OptionID, &v.Provider, &v.Model, &v.PostID, &v.Status, &v.Error)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}
func (s *Service) List(ctx context.Context, accountID uint64, limit int) ([]*Item, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+itemColumns+` FROM ai_content_items WHERE ai_account_id=? ORDER BY id DESC LIMIT ?`, accountID, min(max(limit, 1), 50))
	if err != nil {
		return nil, err
	}
	items := []*Item{}
	for rows.Next() {
		item, e := scanItem(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		rows, err = s.DB.QueryContext(ctx, `SELECT v.option_id,v.provider,v.model,v.post_id,IF(p.deleted IS NOT NULL,'removed',v.status),COALESCE(v.error,''),COALESCE(p.body,'') FROM ai_content_variants v LEFT JOIN posts p ON p.id=v.post_id WHERE content_item_id=? ORDER BY option_id`, item.ID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var v Variant
			if err = rows.Scan(&v.OptionID, &v.Provider, &v.Model, &v.PostID, &v.Status, &v.Error, &v.Body); err != nil {
				rows.Close()
				return nil, err
			}
			item.Variants = append(item.Variants, v)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}
