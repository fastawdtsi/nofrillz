package sessions

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const SessionTokenBytes = 32

type AccessTokenClaims struct {
	UserID    uint64 `json:"sub"`
	SessionID uint64 `json:"sid"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

type sessionRepository interface {
	Create(ctx context.Context, sessionID uint64, userID uint64) error
	GetActiveByID(ctx context.Context, sessionID uint64) (*Session, error)
	RevokeByID(ctx context.Context, sessionID uint64) (bool, error)
	CreateRefresh(ctx context.Context, tokenHash string, sessionID uint64, expiresAt time.Time) error
	GetActiveRefreshByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshByTokenHash(ctx context.Context, tokenHash string) (bool, error)
	RevokeRefreshBySessionID(ctx context.Context, sessionID uint64) (bool, error)
}

type Service struct {
	repository sessionRepository
}

func GenerateSessionToken() (string, error) {
	bytes := make([]byte, SessionTokenBytes)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func HashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func NewService(repository sessionRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, sessionID uint64, userID uint64) error {
	return s.repository.Create(ctx, sessionID, userID)
}

func (s *Service) GetActiveByID(ctx context.Context, sessionID uint64) (*Session, error) {
	return s.repository.GetActiveByID(ctx, sessionID)
}

func (s *Service) RevokeByID(ctx context.Context, sessionID uint64) (bool, error) {
	return s.repository.RevokeByID(ctx, sessionID)
}

func (s *Service) CreateRefresh(ctx context.Context, tokenHash string, sessionID uint64, expiresAt time.Time) error {
	return s.repository.CreateRefresh(ctx, tokenHash, sessionID, expiresAt)
}

func (s *Service) GetActiveRefreshByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	return s.repository.GetActiveRefreshByTokenHash(ctx, tokenHash)
}

func (s *Service) RevokeRefreshByTokenHash(ctx context.Context, tokenHash string) (bool, error) {
	return s.repository.RevokeRefreshByTokenHash(ctx, tokenHash)
}

func (s *Service) RevokeRefreshBySessionID(ctx context.Context, sessionID uint64) (bool, error) {
	return s.repository.RevokeRefreshBySessionID(ctx, sessionID)
}

func GenerateAccessToken(secret string, userID uint64, sessionID uint64, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret cannot be empty")
	}

	now := time.Now().UTC()
	claims := AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		ExpiresAt: now.Add(ttl).Unix(),
		IssuedAt:  now.Unix(),
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}

	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHeader + "." + encodedClaims
	signature := signHS256(signingInput, secret)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + encodedSignature, nil
}

func ParseAccessToken(secret string, token string) (*AccessTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt format")
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.New("invalid jwt format")
	}
	expectedSignature := signHS256(signingInput, secret)
	if !hmac.Equal(signature, expectedSignature) {
		return nil, errors.New("invalid jwt signature")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid jwt format")
	}

	claims := AccessTokenClaims{}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, errors.New("invalid jwt claims")
	}
	if claims.UserID == 0 || claims.SessionID == 0 || claims.ExpiresAt <= 0 {
		return nil, errors.New("invalid jwt claims")
	}
	if time.Now().UTC().Unix() >= claims.ExpiresAt {
		return nil, errors.New("jwt expired")
	}

	return &claims, nil
}

func signHS256(payload string, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}
