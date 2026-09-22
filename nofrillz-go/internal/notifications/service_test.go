package notifications

import (
	"context"
	"errors"
	"testing"

	"nofrillz/internal/apns"
)

type notificationsRepoStub struct {
	settings  *Settings
	getErr    error
	upserted  *Settings
	upsertErr error
}

func (s *notificationsRepoStub) GetByUserID(ctx context.Context, userID uint64) (*Settings, error) {
	return s.settings, s.getErr
}

func (s *notificationsRepoStub) Upsert(ctx context.Context, settings *Settings) error {
	copied := *settings
	s.upserted = &copied
	return s.upsertErr
}

type notificationsSenderStub struct {
	summary *apns.SendSummary
	err     error
	userID  uint64
	title   string
	body    string
}

func (s *notificationsSenderStub) SendOneOff(ctx context.Context, userID uint64, title string, body string) (*apns.SendSummary, error) {
	s.userID = userID
	s.title = title
	s.body = body
	return s.summary, s.err
}

func TestGetByUserIDReturnsDefaultsWhenMissing(t *testing.T) {
	service := NewService(nil, &notificationsRepoStub{}, &notificationsSenderStub{})

	settings, err := service.GetByUserID(context.Background(), 77)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !settings.Enabled || !settings.NewFollowers || settings.NewLikes || !settings.Replies {
		t.Fatalf("unexpected default settings: %+v", settings)
	}
}

func TestUpdateUserSettingsPersistsMergedValues(t *testing.T) {
	repo := &notificationsRepoStub{}
	service := NewService(nil, repo, &notificationsSenderStub{})
	newLikes := true
	enabled := false

	settings, err := service.UpdateUserSettings(context.Background(), 88, UpdateSettingsInput{
		Enabled:  &enabled,
		NewLikes: &newLikes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if settings.Enabled != false || settings.NewFollowers != true || settings.NewLikes != true || settings.Replies != true {
		t.Fatalf("unexpected merged settings: %+v", settings)
	}
	if repo.upserted == nil || repo.upserted.UserID != 88 {
		t.Fatalf("expected settings to be upserted for user 88")
	}
}

func TestNotifyNewFollowerHonorsSettings(t *testing.T) {
	sender := &notificationsSenderStub{}
	service := NewService(nil, &notificationsRepoStub{settings: &Settings{UserID: 9, Enabled: true, NewFollowers: false}}, sender)

	if err := service.NotifyNewFollower(context.Background(), 9, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sender.userID != 0 {
		t.Fatalf("expected notification to be skipped")
	}
}

func TestNotifyNewFollowerSendsExpectedMessage(t *testing.T) {
	sender := &notificationsSenderStub{summary: &apns.SendSummary{Total: 1, Success: 1}}
	service := NewService(nil, &notificationsRepoStub{}, sender)

	if err := service.NotifyNewFollower(context.Background(), 10, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sender.userID != 10 || sender.title != "New follower" || sender.body != "alice followed you." {
		t.Fatalf("unexpected send payload: user=%d title=%q body=%q", sender.userID, sender.title, sender.body)
	}
}

func TestNotifyNewFollowerReturnsSenderError(t *testing.T) {
	expectedErr := errors.New("send failed")
	service := NewService(nil, &notificationsRepoStub{}, &notificationsSenderStub{err: expectedErr})

	if err := service.NotifyNewFollower(context.Background(), 11, "alice"); !errors.Is(err, expectedErr) {
		t.Fatalf("expected sender error, got %v", err)
	}
}
