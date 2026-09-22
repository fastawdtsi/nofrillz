package users

import "testing"

func TestGenerateAndValidatePassword(t *testing.T) {
	hash, salt, err := GeneratePasswordHashAndSalt("supersecret")
	if err != nil {
		t.Fatalf("GeneratePasswordHashAndSalt: %v", err)
	}

	if len(hash) == 0 {
		t.Fatalf("expected non-empty hash")
	}
	if len(salt) == 0 {
		t.Fatalf("expected non-empty salt")
	}

	if !ValidatePassword("supersecret", hash, salt) {
		t.Fatalf("expected password validation to succeed")
	}

	if ValidatePassword("wrongpassword", hash, salt) {
		t.Fatalf("expected password validation to fail for wrong password")
	}
}

func TestGeneratePasswordHashAndSalt_EmptyPassword(t *testing.T) {
	_, _, err := GeneratePasswordHashAndSalt("")
	if err == nil {
		t.Fatalf("expected error for empty password")
	}
}
