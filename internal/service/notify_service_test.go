package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
)

// mockEmailSender implements EmailSender for testing.
type mockEmailSender struct {
	SendFeedbackDigestFn func(ctx context.Context, toName, toEmail string, count int) error
	calls                []emailCall
}

type emailCall struct {
	toName  string
	toEmail string
	count   int
}

func (m *mockEmailSender) SendFeedbackDigest(ctx context.Context, toName, toEmail string, count int) error {
	m.calls = append(m.calls, emailCall{toName: toName, toEmail: toEmail, count: count})
	if m.SendFeedbackDigestFn != nil {
		return m.SendFeedbackDigestFn(ctx, toName, toEmail, count)
	}
	return nil
}

func (m *mockEmailSender) SendPasswordResetEmail(ctx context.Context, toName, toEmail, resetLink string) error {
	return nil
}

// helpers

var (
	alice = &model.User{ID: "u1", Name: "Alice", Email: "alice@example.com", TeamID: "team-a"}
	bob   = &model.User{ID: "u2", Name: "Bob", Email: "bob@example.com", TeamID: "team-a"}
	carol = &model.User{ID: "u3", Name: "Carol", Email: "carol@example.com", TeamID: "team-b"}
)

func oneFeedback() []*model.Feedback { return []*model.Feedback{{ID: "f1"}} }
func noFeedback() []*model.Feedback  { return nil }

// allUsersRepo returns a mockUserRepository with all three test users.
func allUsersRepo() *mockUserRepository {
	return &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			return []*model.User{alice, bob, carol}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, teamID string) ([]*model.User, error) {
			switch teamID {
			case "team-a":
				return []*model.User{alice, bob}, nil
			case "team-b":
				return []*model.User{carol}, nil
			default:
				return []*model.User{}, nil
			}
		},
	}
}

// alwaysFeedbackRepo returns a feedback repo where every user has one new feedback.
func alwaysFeedbackRepo() *mockFeedbackRepository {
	return &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, _ string, _ time.Time) ([]*model.Feedback, error) {
			return oneFeedback(), nil
		},
	}
}

// --- All-users (empty teamID) ---

func TestSendFeedbackDigest_AllUsers_NotifiesUsersWithNewFeedback(t *testing.T) {
	feedbackRepo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, revieweeID string, _ time.Time) ([]*model.Feedback, error) {
			if revieweeID == "u1" {
				return oneFeedback(), nil
			}
			return noFeedback(), nil
		},
	}

	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), feedbackRepo, sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Notified)
	assert.Equal(t, 2, result.Skipped)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, "Alice", sender.calls[0].toName)
	assert.Equal(t, "alice@example.com", sender.calls[0].toEmail)
}

func TestSendFeedbackDigest_AllUsers_CountsMultipleFeedbacks(t *testing.T) {
	feedbackRepo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, revieweeID string, _ time.Time) ([]*model.Feedback, error) {
			if revieweeID == "u1" {
				return []*model.Feedback{{ID: "f1"}, {ID: "f2"}, {ID: "f3"}}, nil
			}
			return noFeedback(), nil
		},
	}

	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), feedbackRepo, sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Notified)
	assert.Equal(t, 3, sender.calls[0].count)
}

func TestSendFeedbackDigest_AllUsers_GetAllUsersError(t *testing.T) {
	userRepo := &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := NewNotifyService(userRepo, &mockFeedbackRepository{}, &mockEmailSender{})
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSendFeedbackDigest_AllUsers_FeedbackRepoError_SkipsUser(t *testing.T) {
	feedbackRepo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, revieweeID string, _ time.Time) ([]*model.Feedback, error) {
			if revieweeID == "u1" {
				return nil, fmt.Errorf("repo error")
			}
			return oneFeedback(), nil
		},
	}
	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), feedbackRepo, sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	// Alice skipped (repo error), Bob and Carol notified
	assert.Equal(t, 2, result.Notified)
	assert.Equal(t, 1, result.Skipped)
}

func TestSendFeedbackDigest_AllUsers_EmailSendError_SkipsUser(t *testing.T) {
	sender := &mockEmailSender{
		SendFeedbackDigestFn: func(_ context.Context, _, toEmail string, _ int) error {
			if toEmail == "alice@example.com" {
				return fmt.Errorf("smtp error")
			}
			return nil
		},
	}
	svc := NewNotifyService(allUsersRepo(), alwaysFeedbackRepo(), sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, 2, result.Notified) // Bob + Carol succeed
	assert.Equal(t, 1, result.Skipped)  // Alice's send failed
}

func TestSendFeedbackDigest_AllUsers_NoUsers(t *testing.T) {
	userRepo := &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			return []*model.User{}, nil
		},
	}
	svc := NewNotifyService(userRepo, &mockFeedbackRepository{}, &mockEmailSender{})
	result, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Notified)
	assert.Equal(t, 0, result.Skipped)
}

func TestSendFeedbackDigest_AllUsers_SinceIsStartOfYesterday(t *testing.T) {
	var capturedSince time.Time
	userRepo := &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			return []*model.User{alice}, nil
		},
	}
	feedbackRepo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, _ string, since time.Time) ([]*model.Feedback, error) {
			capturedSince = since
			return noFeedback(), nil
		},
	}
	svc := NewNotifyService(userRepo, feedbackRepo, &mockEmailSender{})
	_, err := svc.SendFeedbackDigest(context.Background(), "")
	require.NoError(t, err)

	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	expected := startOfToday.AddDate(0, 0, -1)
	assert.Equal(t, expected, capturedSince)
}

// --- Team-scoped ---

func TestSendFeedbackDigest_TeamScoped_OnlyNotifiesTeamMembers(t *testing.T) {
	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), alwaysFeedbackRepo(), sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "team-a")

	require.NoError(t, err)
	assert.Equal(t, 2, result.Notified) // Alice + Bob only
	assert.Equal(t, 0, result.Skipped)

	notifiedNames := make([]string, len(sender.calls))
	for i, c := range sender.calls {
		notifiedNames[i] = c.toName
	}
	assert.ElementsMatch(t, []string{"Alice", "Bob"}, notifiedNames)
}

func TestSendFeedbackDigest_TeamScoped_SingleTeamMember(t *testing.T) {
	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), alwaysFeedbackRepo(), sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "team-b")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Notified) // Carol only
	require.Len(t, sender.calls, 1)
	assert.Equal(t, "Carol", sender.calls[0].toName)
}

func TestSendFeedbackDigest_TeamScoped_TeamNotFound_ReturnsEmpty(t *testing.T) {
	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), alwaysFeedbackRepo(), sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "team-unknown")

	require.NoError(t, err)
	assert.Equal(t, 0, result.Notified)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, sender.calls)
}

func TestSendFeedbackDigest_TeamScoped_RepoError_ReturnsError(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	svc := NewNotifyService(userRepo, &mockFeedbackRepository{}, &mockEmailSender{})
	result, err := svc.SendFeedbackDigest(context.Background(), "team-a")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSendFeedbackDigest_TeamScoped_SkipMembersWithNoNewFeedback(t *testing.T) {
	feedbackRepo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDSinceFn: func(_ context.Context, revieweeID string, _ time.Time) ([]*model.Feedback, error) {
			if revieweeID == "u1" { // Alice
				return oneFeedback(), nil
			}
			return noFeedback(), nil // Bob has none
		},
	}
	sender := &mockEmailSender{}
	svc := NewNotifyService(allUsersRepo(), feedbackRepo, sender)
	result, err := svc.SendFeedbackDigest(context.Background(), "team-a")

	require.NoError(t, err)
	assert.Equal(t, 1, result.Notified)
	assert.Equal(t, 1, result.Skipped)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, "Alice", sender.calls[0].toName)
}

func TestSendFeedbackDigest_TeamScoped_UsesGetUsersByTeamID_NotGetAllUsers(t *testing.T) {
	getAllCalled := false
	getByTeamCalled := false

	userRepo := &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			getAllCalled = true
			return nil, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			getByTeamCalled = true
			return []*model.User{alice}, nil
		},
	}
	svc := NewNotifyService(userRepo, alwaysFeedbackRepo(), &mockEmailSender{})
	_, err := svc.SendFeedbackDigest(context.Background(), "team-a")

	require.NoError(t, err)
	assert.False(t, getAllCalled, "GetAllUsers should not be called when teamID is set")
	assert.True(t, getByTeamCalled, "GetUsersByTeamID should be called when teamID is set")
}

func TestSendFeedbackDigest_AllUsers_UsesGetAllUsers_NotGetUsersByTeamID(t *testing.T) {
	getAllCalled := false
	getByTeamCalled := false

	userRepo := &mockUserRepository{
		GetAllUsersFn: func(_ context.Context) ([]*model.User, error) {
			getAllCalled = true
			return []*model.User{alice}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			getByTeamCalled = true
			return nil, nil
		},
	}
	svc := NewNotifyService(userRepo, alwaysFeedbackRepo(), &mockEmailSender{})
	_, err := svc.SendFeedbackDigest(context.Background(), "")

	require.NoError(t, err)
	assert.True(t, getAllCalled, "GetAllUsers should be called when teamID is empty")
	assert.False(t, getByTeamCalled, "GetUsersByTeamID should not be called when teamID is empty")
}
