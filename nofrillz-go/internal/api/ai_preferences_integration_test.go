package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aitools"
	"nofrillz/internal/app"
	"nofrillz/internal/db"
	"nofrillz/internal/limiters"
	"nofrillz/internal/testdb"
)

func TestModelPreferenceAPI(t *testing.T) {
	database := testdb.Open(t)
	for _, id := range []uint64{1, 10, 11} {
		_, err := database.Exec(`INSERT INTO users(id,email,username,first_name,last_name,about,account_type,password_hash) VALUES(?,?,?,?,?,?,'ai',?)`, id, fmt.Sprintf("test%d@example.invalid", id), fmt.Sprintf("test%d", id), "Content", "", "About", []byte("unused"))
		if err != nil {
			t.Fatal(err)
		}
	}
	models, err := aitools.NewRegistry([]aitools.Option{{ID: "openai", Provider: "openai", Model: "test-a", Tools: aitools.NewMockTools()}, {ID: "claude", Provider: "anthropic", Model: "test-b", Tools: aitools.NewMockTools()}, {ID: "grok", Provider: "xai", Model: "test-c"}})
	if err != nil {
		t.Fatal(err)
	}
	accounts := aiaccounts.NewRepository(database)
	if err = accounts.Create(context.Background(), &aiaccounts.AIAccount{ID: 2, UserID: 1, Description: "Mission", Topic: "Topic", Enabled: true, ContentMode: "generative", ModelOptions: []string{"openai", "claude"}, DefaultModelOption: "openai", MinPostsPerDay: 1, MaxPostsPerDay: 2}); err != nil {
		t.Fatal(err)
	}
	logger := zerolog.Nop()
	a := &app.App{MySQL: &db.MySQL{DB: database}, AIModels: models, AIAccounts: aiaccounts.NewService(accounts), Logger: &logger, Limiters: limiters.NewLimiters(nil)}
	mux := http.NewServeMux()
	NewAIModelHandler(a).AddRoutes(mux, func(h http.HandlerFunc) http.HandlerFunc { return h })
	request := func(user uint64, method, path, body string, want int) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != 0 {
			req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: user}))
		}
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != want {
			t.Fatalf("%s %s: %d, %s", method, path, rr.Code, rr.Body.String())
		}
		out := map[string]any{}
		if want == 200 {
			if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	request(0, "GET", "/ai/models", "", 200)
	request(0, "GET", "/users/me/ai-preference", "", 401)
	request(0, "PATCH", "/ai/accounts/1/preference", `{"model_option":"claude"}`, 401)
	request(0, "GET", "/posts/999/ai-content", "", 401)
	request(10, "PATCH", "/users/me/ai-preference", `{"model_option":"unknown"}`, 400)
	got := request(10, "PATCH", "/users/me/ai-preference", `{"model_option":"claude"}`, 200)
	if got["model_option"] != "claude" {
		t.Fatal("global preference not persisted")
	}
	request(10, "PATCH", "/ai/accounts/1/preference", `{"model_option":"openai"}`, 409)
	if _, err := database.Exec(`INSERT INTO follows(follower_id,following_id) VALUES(10,1)`); err != nil {
		t.Fatal(err)
	}
	got = request(10, "GET", "/ai/accounts/1/models", "", 200)
	if got["effective_model_option"] != "claude" {
		t.Fatal("global preference ignored")
	}
	got = request(10, "PATCH", "/ai/accounts/1/preference", `{"model_option":"openai"}`, 200)
	if got["effective_model_option"] != "openai" {
		t.Fatal("override ignored")
	}
	request(11, "PATCH", "/ai/accounts/1/preference", `{"model_option":"claude"}`, 409)
	got = request(10, "GET", "/ai/accounts/1/models", "", 200)
	if got["override"] != "openai" {
		t.Fatal("another user's request modified override")
	}
	got = request(10, "PATCH", "/ai/accounts/1/preference", `{"model_option":"grok"}`, 200)
	if got["effective_model_option"] != "claude" {
		t.Fatal("unavailable override did not fall back")
	}
	got = request(10, "DELETE", "/ai/accounts/1/preference", "", 200)
	if got["override"] != nil || got["effective_model_option"] != "claude" {
		t.Fatal("override reset failed")
	}
	got = request(10, "PATCH", "/users/me/ai-preference", `{"model_option":null}`, 200)
	if got["model_option"] != nil {
		t.Fatal("global preference reset failed")
	}
	got = request(10, "GET", "/ai/accounts/1/models", "", 200)
	if got["effective_model_option"] != "openai" {
		t.Fatal("account default fallback failed")
	}
}
