package users

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
)

const (
	passwordSaltBytes  = 16
	passwordHashBytes  = 32
	passwordHashRounds = 100000
)

func GeneratePasswordHashAndSalt(password string) ([]byte, []byte, error) {
	if password == "" {
		return nil, nil, errors.New("password cannot be empty")
	}

	salt := make([]byte, passwordSaltBytes)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, nil, err
	}

	hash := derivePasswordHash(password, salt)

	return hash, salt, nil
}

func ValidatePassword(password string, expectedHash []byte, salt []byte) bool {
	if password == "" || len(expectedHash) == 0 || len(salt) == 0 {
		return false
	}

	hash := derivePasswordHash(password, salt)

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

func derivePasswordHash(password string, salt []byte) []byte {
	data := append([]byte(password), salt...)
	digest := sha256.Sum256(data)

	hash := digest[:]
	for i := 0; i < passwordHashRounds; i++ {
		roundData := append(hash, salt...)
		nextDigest := sha256.Sum256(roundData)
		hash = nextDigest[:]
	}

	output := make([]byte, passwordHashBytes)
	copy(output, hash[:passwordHashBytes])

	return output
}
