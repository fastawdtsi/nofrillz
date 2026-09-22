package apns

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrInvalidToken  = errors.New("invalid apns device token")
	ErrInvalidTitle  = errors.New("invalid title")
	ErrInvalidBody   = errors.New("invalid body")
)

type deviceTokenRepository interface {
	ListByUserID(ctx context.Context, userID uint64) ([]*DeviceToken, error)
	Upsert(ctx context.Context, userID uint64, token string) error
	DeleteTokens(ctx context.Context, tokens []string) error
}

type Service struct {
	logger     *zerolog.Logger
	repository deviceTokenRepository
	provider   deliveryProvider
}

func NewService(logger *zerolog.Logger, repository deviceTokenRepository, provider deliveryProvider) *Service {
	return &Service{logger: logger, repository: repository, provider: provider}
}

func NormalizeToken(raw string) (string, error) {
	token := strings.ToLower(strings.TrimSpace(raw))
	if token == "" || len(token) > maxTokenLength {
		return "", ErrInvalidToken
	}
	if _, err := hex.DecodeString(token); err != nil {
		return "", ErrInvalidToken
	}

	return token, nil
}

func (s *Service) RegisterToken(ctx context.Context, userID uint64, rawToken string) (string, error) {
	if userID == 0 {
		return "", ErrInvalidUserID
	}

	token, err := NormalizeToken(rawToken)
	if err != nil {
		return "", err
	}

	if err := s.repository.Upsert(ctx, userID, token); err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) SendOneOff(ctx context.Context, userID uint64, title string, body string) (*SendSummary, error) {
	if userID == 0 {
		return nil, ErrInvalidUserID
	}
	if strings.TrimSpace(title) == "" {
		return nil, ErrInvalidTitle
	}
	if strings.TrimSpace(body) == "" {
		return nil, ErrInvalidBody
	}

	deviceTokens, err := s.repository.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := &SendSummary{Total: len(deviceTokens)}
	if len(deviceTokens) == 0 {
		return summary, nil
	}

	invalidTokens := []string{}
	errs := []error{}
	for _, deviceToken := range deviceTokens {
		result, sendErr := s.provider.Send(ctx, deviceToken.Token, title, body)
		if sendErr != nil {
			if result != nil && result.InvalidToken {
				summary.Invalid++
				invalidTokens = append(invalidTokens, deviceToken.Token)
				if s.logger != nil {
					s.logger.Info().Uint64("user_id", userID).Str("token", deviceToken.Token).Str("reason", result.Reason).Msg("removing invalid apns device token")
				}
				continue
			}
			summary.Failed++
			errs = append(errs, fmt.Errorf("send apns notification to token %s: %w", deviceToken.Token, sendErr))
			continue
		}

		summary.Success++
	}

	if err := s.repository.DeleteTokens(ctx, invalidTokens); err != nil {
		errs = append(errs, fmt.Errorf("delete invalid apns tokens: %w", err))
	}

	if len(errs) > 0 {
		return summary, errors.Join(errs...)
	}

	return summary, nil
}
