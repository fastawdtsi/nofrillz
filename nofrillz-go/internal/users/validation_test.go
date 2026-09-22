package users

import "testing"

func TestValidateEmail(t *testing.T) {
	validEmails := []string{
		"user@example.com",
		"first.last+tag@sub.example.co",
		"a_b-c.d@domain.io",
	}

	for _, email := range validEmails {
		if !ValidateEmail(email) {
			t.Fatalf("expected valid email: %q", email)
		}
	}

	invalidEmails := []string{
		"",
		" user@example.com",
		"user@example.com ",
		"user@@example.com",
		"userexample.com",
		"Name <user@example.com>",
	}

	for _, email := range invalidEmails {
		if ValidateEmail(email) {
			t.Fatalf("expected invalid email: %q", email)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	validUsernames := []string{
		"abc",
		"Alice_1",
		"user-name",
		"first.last",
		"Z9_-.__",
	}

	for _, username := range validUsernames {
		if !ValidateUsername(username) {
			t.Fatalf("expected valid username: %q", username)
		}
	}

	invalidUsernames := []string{
		"",
		"ab",
		" user",
		"user ",
		"user name",
		"user@name",
		"üser",
		"名字",
	}

	for _, username := range invalidUsernames {
		if ValidateUsername(username) {
			t.Fatalf("expected invalid username: %q", username)
		}
	}
}
