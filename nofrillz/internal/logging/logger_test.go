package logging

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevelDefaultsToInfo(t *testing.T) {
	level, err := ParseLevel("")
	if err != nil {
		t.Fatalf("ParseLevel: %v", err)
	}
	if level != zerolog.InfoLevel {
		t.Fatalf("expected info level, got %s", level.String())
	}
}

func TestParseLevelSupportsDebugInfoWarn(t *testing.T) {
	tests := map[string]zerolog.Level{
		"debug": zerolog.DebugLevel,
		"info":  zerolog.InfoLevel,
		"warn":  zerolog.WarnLevel,
	}

	for input, expected := range tests {
		level, err := ParseLevel(input)
		if err != nil {
			t.Fatalf("ParseLevel(%q): %v", input, err)
		}
		if level != expected {
			t.Fatalf("ParseLevel(%q): expected %s, got %s", input, expected.String(), level.String())
		}
	}
}

func TestParseLevelRejectsUnsupportedValues(t *testing.T) {
	level, err := ParseLevel("verbose")
	if err == nil {
		t.Fatalf("expected error for unsupported level")
	}
	if level != zerolog.InfoLevel {
		t.Fatalf("expected fallback info level, got %s", level.String())
	}
}
