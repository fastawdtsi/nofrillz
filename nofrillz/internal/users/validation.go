package users

import (
	"net/mail"
	"regexp"
	"strings"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 32
	MaxEmailLength    = 320
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func ValidateEmail(email string) bool {
	if email == "" || len(email) > MaxEmailLength {
		return false
	}
	if strings.TrimSpace(email) != email {
		return false
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	return parsed.Address == email
}

func ValidateUsername(username string) bool {
	length := len(username)
	if length < MinUsernameLength || length > MaxUsernameLength {
		return false
	}
	if strings.TrimSpace(username) != username {
		return false
	}

	return usernamePattern.MatchString(username)
}
