package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nofrillz/internal/bookmarks"
)

func TestIDCursorEncodeDecode(t *testing.T) {
	original := uint64(123456789)

	encoded, err := EncodeIDCursor(&original)
	if err != nil {
		t.Fatalf("EncodeIDCursor: %v", err)
	}
	if encoded == "" {
		t.Fatalf("expected non-empty encoded cursor")
	}

	decoded, err := DecodeIDCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeIDCursor: %v", err)
	}
	if decoded == nil {
		t.Fatalf("expected decoded cursor")
	}
	if *decoded != original {
		t.Fatalf("cursor mismatch got=%d want=%d", *decoded, original)
	}
}

func TestIDCursorDecodeInvalid(t *testing.T) {
	_, err := DecodeIDCursor("%%%")
	if err == nil {
		t.Fatalf("expected error for invalid cursor")
	}
}

func TestBookmarkCursorEncodeDecode(t *testing.T) {
	original := &bookmarks.Cursor{
		Created: time.Date(2026, 7, 11, 18, 0, 0, 123456000, time.UTC),
		PostID:  987654321,
	}

	encoded, err := EncodeBookmarkCursor(original)
	if err != nil {
		t.Fatalf("EncodeBookmarkCursor: %v", err)
	}
	if encoded == "" {
		t.Fatalf("expected non-empty encoded cursor")
	}

	decoded, err := DecodeBookmarkCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeBookmarkCursor: %v", err)
	}
	if decoded == nil {
		t.Fatalf("expected decoded cursor")
	}
	if !decoded.Created.Equal(original.Created) || decoded.PostID != original.PostID {
		t.Fatalf("cursor mismatch got=%+v want=%+v", *decoded, *original)
	}
}

func TestBookmarkCursorDecodeInvalid(t *testing.T) {
	_, err := DecodeBookmarkCursor("%%%")
	if err == nil {
		t.Fatalf("expected error for invalid bookmark cursor")
	}
}

func TestDecodeRequestBodyUnicode(t *testing.T) {
	type unicodePayload struct {
		Body string `json:"body"`
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/test",
		bytes.NewBufferString(`{"body":"hello 👋 こんにちは مرحبا 你好"}`),
	)
	rr := httptest.NewRecorder()

	decoded, err := DecodeRequestBody[unicodePayload](rr, req, 1<<20)
	if err != nil {
		t.Fatalf("DecodeRequestBody: %v", err)
	}

	want := "hello 👋 こんにちは مرحبا 你好"
	if decoded.Body != want {
		t.Fatalf("unicode mismatch got=%q want=%q", decoded.Body, want)
	}
}

func TestRespondUnicode(t *testing.T) {
	type unicodeResponse struct {
		Body string `json:"body"`
	}

	rr := httptest.NewRecorder()
	Respond(rr, http.StatusOK, unicodeResponse{Body: "café ☕ привет नमस्ते"}, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	var decoded unicodeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	want := "café ☕ привет नमस्ते"
	if decoded.Body != want {
		t.Fatalf("unicode mismatch got=%q want=%q", decoded.Body, want)
	}
}
