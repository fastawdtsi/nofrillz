package notifications

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"nofrillz/internal/apns"
)

var ErrInvalidUserID = errors.New("invalid user id")

type settingsRepository interface {
	GetByUserID(ctx context.Context, userID uint64) (*Settings, error)
	Upsert(ctx context.Context, settings *Settings) error
}

type pushSender interface {
	SendOneOff(ctx context.Context, userID uint64, title string, body string) (*apns.SendSummary, error)
}

type Service struct {
	logger     *zerolog.Logger
	repository settingsRepository
	sender     pushSender
}

func NewService(logger *zerolog.Logger, repository settingsRepository, sender pushSender) *Service {
	return &Service{logger: logger, repository: repository, sender: sender}
}

func DefaultSettings(userID uint64) *Settings {
	return &Settings{
		UserID:       userID,
		Enabled:      true,
		NewFollowers: true,
		NewLikes:     false,
		Replies:      true,
	}
}

func (s *Service) GetByUserID(ctx context.Context, userID uint64) (*Settings, error) {
	if userID == 0 {
		return nil, ErrInvalidUserID
	}

	settings, err := s.repository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return DefaultSettings(userID), nil
	}

	return settings, nil
}

func (s *Service) UpdateUserSettings(ctx context.Context, userID uint64, input UpdateSettingsInput) (*Settings, error) {
	settings, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.Enabled != nil {
		settings.Enabled = *input.Enabled
	}
	if input.NewFollowers != nil {
		settings.NewFollowers = *input.NewFollowers
	}
	if input.NewLikes != nil {
		settings.NewLikes = *input.NewLikes
	}
	if input.Replies != nil {
		settings.Replies = *input.Replies
	}

	if err := s.repository.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (s *Service) NotifyNewFollower(ctx context.Context, userID uint64, followerLabel string) error {
	settings, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !settings.Enabled || !settings.NewFollowers {
		if s.logger != nil {
			s.logger.Debug().Uint64("user_id", userID).Msg("skipping new follower push notification due to user settings")
		}
		return nil
	}

	message := "Someone followed you."
	if followerLabel = strings.TrimSpace(followerLabel); followerLabel != "" {
		message = followerLabel + " followed you."
	}

	summary, err := s.sender.SendOneOff(ctx, userID, "New follower", message)
	if err != nil {
		return fmt.Errorf("send new follower push notification: %w", err)
	}
	if s.logger != nil && summary != nil {
		s.logger.Info().
			Uint64("user_id", userID).
			Int("total_tokens", summary.Total).
			Int("success", summary.Success).
			Int("invalid", summary.Invalid).
			Int("failed", summary.Failed).
			Msg("sent new follower push notification")
	}

	return nil
}
