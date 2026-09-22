package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"nofrillz/internal/apns"
	"nofrillz/internal/app"
	"nofrillz/internal/follows"
	"nofrillz/internal/limiters"
	"nofrillz/internal/notifications"
)

type apiUsersFollowRepoStub struct {
	created     bool
	followErr   error
	followerID  uint64
	followingID uint64
}

func (s *apiUsersFollowRepoStub) Follow(ctx context.Context, followerID uint64, followingID uint64) (bool, error) {
	s.followerID = followerID
	s.followingID = followingID
	return s.created, s.followErr
}

func (s *apiUsersFollowRepoStub) Unfollow(ctx context.Context, followerID uint64, followingID uint64) error {
	return nil
}

func (s *apiUsersFollowRepoStub) ListFollowers(ctx context.Context, userID uint64) ([]uint64, error) {
	return nil, nil
}

func (s *apiUsersFollowRepoStub) ListFollowersPage(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*follows.Follower, error) {
	return nil, nil
}

func (s *apiUsersFollowRepoStub) ListFollowing(ctx context.Context, userID uint64) ([]uint64, error) {
	return nil, nil
}

type apiUsersNotificationsRepoStub struct{}

func (s *apiUsersNotificationsRepoStub) GetByUserID(ctx context.Context, userID uint64) (*notifications.Settings, error) {
	return nil, nil
}

func (s *apiUsersNotificationsRepoStub) Upsert(ctx context.Context, settings *notifications.Settings) error {
	return nil
}

type apiUsersSenderStub struct {
	userID uint64
	title  string
	body   string
}

func (s *apiUsersSenderStub) SendOneOff(ctx context.Context, userID uint64, title string, body string) (*apns.SendSummary, error) {
	s.userID = userID
	s.title = title
	s.body = body
	return &apns.SendSummary{Total: 1, Success: 1}, nil
}

func TestUsersHandlerFollowSendsNotificationForNewFollow(t *testing.T) {
	followRepo := &apiUsersFollowRepoStub{created: true}
	sender := &apiUsersSenderStub{}
	logger := zerolog.Nop()
	handler := NewUsersHandler(&app.App{
		Follows:       follows.NewService(followRepo),
		Notifications: notifications.NewService(&logger, &apiUsersNotificationsRepoStub{}, sender),
		Limiters:      limiters.NewLimiters(nil),
		Logger:        &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/users/55/follow", nil)
	req.SetPathValue("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44, Username: "alice"}))
	rr := httptest.NewRecorder()

	handler.Follow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}
	if sender.userID != 55 || sender.title != "New follower" || sender.body != "alice followed you." {
		t.Fatalf("unexpected notification payload: user=%d title=%q body=%q", sender.userID, sender.title, sender.body)
	}
}

func TestUsersHandlerFollowDoesNotSendNotificationForExistingFollow(t *testing.T) {
	followRepo := &apiUsersFollowRepoStub{created: false}
	sender := &apiUsersSenderStub{}
	logger := zerolog.Nop()
	handler := NewUsersHandler(&app.App{
		Follows:       follows.NewService(followRepo),
		Notifications: notifications.NewService(&logger, &apiUsersNotificationsRepoStub{}, sender),
		Limiters:      limiters.NewLimiters(nil),
		Logger:        &logger,
	})

	req := httptest.NewRequest(http.MethodPost, "/users/55/follow", nil)
	req.SetPathValue("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44, Username: "alice"}))
	rr := httptest.NewRecorder()

	handler.Follow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}
	if sender.userID != 0 {
		t.Fatalf("expected notification not to be sent for existing follow")
	}
}

func TestUsersHandlerPushNotificationSettingsDefaults(t *testing.T) {
	logger := zerolog.Nop()
	handler := NewUsersHandler(&app.App{
		Notifications: notifications.NewService(&logger, &apiUsersNotificationsRepoStub{}, &apiUsersSenderStub{}),
		Limiters:      limiters.NewLimiters(nil),
		Logger:        &logger,
	})

	req := httptest.NewRequest(http.MethodGet, "/users/push-notification-settings", nil)
	req = req.WithContext(context.WithValue(req.Context(), authenticatedUserKey, AuthenticatedUser{ID: 44}))
	rr := httptest.NewRecorder()

	handler.ShowPushNotificationSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["enabled"] != true || body["new_followers"] != true || body["new_likes"] != false || body["replies"] != true {
		t.Fatalf("unexpected default push settings payload: %#v", body)
	}
}
