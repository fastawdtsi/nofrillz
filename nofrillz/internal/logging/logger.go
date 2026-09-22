package logging

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func ParseLevel(raw string) (zerolog.Level, error) {
	level := strings.TrimSpace(strings.ToLower(raw))
	if level == "" {
		return zerolog.InfoLevel, nil
	}

	switch level {
	case "debug", "info", "warn", "error":
		return zerolog.ParseLevel(level)
	default:
		return zerolog.InfoLevel, fmt.Errorf("unsupported log level %q", raw)
	}
}

func NewLogger(component string, rawLevel string) (zerolog.Logger, zerolog.Level, error) {
	level, err := ParseLevel(rawLevel)
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("component", component).Logger().Level(level)
	zerolog.SetGlobalLevel(level)
	return logger, level, err
}
