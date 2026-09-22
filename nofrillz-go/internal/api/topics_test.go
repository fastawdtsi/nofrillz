package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/rs/zerolog"

	"nofrillz/internal/app"
	"nofrillz/internal/limiters"
)

func TestTopicsHandlerListReturnsHardcodedTopics(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewTopicsHandler(&app.App{
		Limiters: limiters.NewLimiters(nil),
		Logger:   &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/topics", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body topicsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if !reflect.DeepEqual(body.Topics, hardcodedTopics()) {
		t.Fatalf("unexpected topics payload: %#v", body.Topics)
	}
}
