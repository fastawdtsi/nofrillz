package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"nofrillz/internal/aitools"
	"nofrillz/internal/app"
	"strconv"
)

type AIModelHandler struct{ app *app.App }

func NewAIModelHandler(a *app.App) *AIModelHandler { return &AIModelHandler{a} }
func (h *AIModelHandler) AddRoutes(m *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	auth := func(f http.HandlerFunc) http.HandlerFunc { return middleware(RequireAuthentication(f)) }
	m.HandleFunc("GET /ai/models", middleware(h.Models))
	m.HandleFunc("GET /users/me/ai-preference", auth(h.Global))
	m.HandleFunc("PATCH /users/me/ai-preference", auth(h.Global))
	m.HandleFunc("GET /ai/accounts/{id}/models", auth(h.Account))
	m.HandleFunc("PATCH /ai/accounts/{id}/preference", auth(h.Account))
	m.HandleFunc("DELETE /ai/accounts/{id}/preference", auth(h.Account))
	m.HandleFunc("GET /posts/{post_id}/ai-content", auth(h.PostContent))
}
func (h *AIModelHandler) Models(w http.ResponseWriter, r *http.Request) {
	Respond(w, 200, map[string]any{"models": h.app.AIModels.Options(), "system_default": "openai"}, h.app.Logger)
}
func (h *AIModelHandler) Global(w http.ResponseWriter, r *http.Request) {
	user, _ := AuthenticatedUserForRequest(r)
	if r.Method == "PATCH" {
		if !AllowWriteByUser(w, h.app, user.ID) {
			return
		}
		req, err := DecodeRequestBody[struct {
			Option *string `json:"model_option"`
		}](w, r, 4096)
		if err != nil {
			http.Error(w, "invalid preference", 400)
			return
		}
		if req.Option != nil {
			if _, ok := h.app.AIModels.Get(*req.Option); !ok {
				http.Error(w, "unknown model option", 400)
				return
			}
		}
		if _, err = h.app.MySQL.DB.ExecContext(r.Context(), `UPDATE users SET ai_model_preference=? WHERE id=?`, req.Option, user.ID); err != nil {
			h.serverError(w, err)
			return
		}
	}
	var preference *string
	if err := h.app.MySQL.DB.QueryRowContext(r.Context(), `SELECT ai_model_preference FROM users WHERE id=?`, user.ID).Scan(&preference); err != nil {
		h.serverError(w, err)
		return
	}
	Respond(w, 200, map[string]any{"model_option": preference, "system_default": "openai"}, h.app.Logger)
}
func (h *AIModelHandler) Account(w http.ResponseWriter, r *http.Request) {
	user, _ := AuthenticatedUserForRequest(r)
	authorID, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid account user id", 400)
		return
	}
	var accountID uint64
	err = h.app.MySQL.DB.QueryRowContext(r.Context(), `SELECT a.id FROM ai_accounts a JOIN users u ON u.id=a.user_id WHERE a.user_id=? AND u.deleted IS NULL AND u.blocked IS NULL`, authorID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(404)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	account, err := h.app.AIAccounts.GetByID(r.Context(), accountID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	var override *string
	err = h.app.MySQL.DB.QueryRowContext(r.Context(), `SELECT ai_model_override FROM follows WHERE follower_id=? AND following_id=?`, user.ID, authorID).Scan(&override)
	following := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.serverError(w, err)
		return
	}
	if r.Method != "GET" {
		if !following {
			http.Error(w, "follow this AI account before setting an override", 409)
			return
		}
		if !AllowWriteByUser(w, h.app, user.ID) {
			return
		}
		override = nil
		if r.Method == "PATCH" {
			req, e := DecodeRequestBody[struct {
				Option *string `json:"model_option"`
			}](w, r, 4096)
			if e != nil {
				http.Error(w, "invalid preference", 400)
				return
			}
			override = req.Option
			if override != nil {
				if _, ok := h.app.AIModels.Get(*override); !ok {
					http.Error(w, "unknown model option", 400)
					return
				}
			}
		}
		if _, err = h.app.MySQL.DB.ExecContext(r.Context(), `UPDATE follows SET ai_model_override=? WHERE follower_id=? AND following_id=?`, override, user.ID, authorID); err != nil {
			h.serverError(w, err)
			return
		}
	}
	var global *string
	if err = h.app.MySQL.DB.QueryRowContext(r.Context(), `SELECT ai_model_preference FROM users WHERE id=?`, user.ID).Scan(&global); err != nil {
		h.serverError(w, err)
		return
	}
	options := []aitools.Option{}
	enabled := map[string]bool{}
	for _, id := range account.ModelOptions {
		if o, ok := h.app.AIModels.Get(id); ok {
			options = append(options, o)
			enabled[id] = o.Available
		}
	}
	order := []string{}
	if override != nil {
		order = append(order, *override)
	}
	if global != nil {
		order = append(order, *global)
	}
	order = append(order, account.DefaultModelOption)
	effective := ""
	for _, id := range order {
		if enabled[id] {
			effective = id
			break
		}
	}
	if effective == "" {
		for _, o := range h.app.AIModels.Options() {
			if enabled[o.ID] {
				effective = o.ID
				break
			}
		}
	}
	Respond(w, 200, map[string]any{"account_user_id": strconv.FormatUint(authorID, 10), "following": following, "override": override, "global_preference": global, "default_model_option": account.DefaultModelOption, "effective_model_option": effective, "models": options, "selection_order": order, "fallback": "first published option in lexical ID order; resolved per content item"}, h.app.Logger)
}
func (h *AIModelHandler) PostContent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("post_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", 400)
		return
	}
	var itemID uint64
	var sources []byte
	var option, provider, model string
	err = h.app.MySQL.DB.QueryRowContext(r.Context(), `SELECT ci.id,ci.sources,v.option_id,v.provider,v.model FROM ai_content_variants v JOIN ai_content_items ci ON ci.id=v.content_item_id JOIN posts p ON p.id=v.post_id AND p.deleted IS NULL JOIN users u ON u.id=p.user_id AND u.deleted IS NULL AND u.blocked IS NULL WHERE v.post_id=?`, id).Scan(&itemID, &sources, &option, &provider, &model)
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(404)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	Respond(w, 200, map[string]any{"content_item_id": strconv.FormatUint(itemID, 10), "model_option": option, "provider": provider, "model": model, "sources": json.RawMessage(sources)}, h.app.Logger)
}
func (h *AIModelHandler) serverError(w http.ResponseWriter, err error) {
	h.app.Logger.Error().Err(err).Msg("AI model preference request failed")
	w.WriteHeader(500)
}
