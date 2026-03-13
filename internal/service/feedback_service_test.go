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

// mockFeedbackRepository implements FeedbackRepository for testing.
type mockFeedbackRepository struct {
	CreateFeedbackFn              func(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedbackFn                 func(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error)
	GetFeedbacksByRevieweeIDFn    func(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDFn    func(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDsFn   func(ctx context.Context, reviewerIDs []string) ([]*model.Feedback, error)
}

func (m *mockFeedbackRepository) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	return m.CreateFeedbackFn(ctx, feedback)
}

func (m *mockFeedbackRepository) GetFeedback(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error) {
	return m.GetFeedbackFn(ctx, reviewerID, revieweeID, period)
}

func (m *mockFeedbackRepository) GetFeedbacksByRevieweeID(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error) {
	return m.GetFeedbacksByRevieweeIDFn(ctx, revieweeID, limit, cursor)
}

func (m *mockFeedbackRepository) GetFeedbacksByReviewerID(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error) {
	return m.GetFeedbacksByReviewerIDFn(ctx, reviewerID, limit, cursor)
}

func (m *mockFeedbackRepository) GetFeedbacksByReviewerIDs(ctx context.Context, reviewerIDs []string) ([]*model.Feedback, error) {
	if m.GetFeedbacksByReviewerIDsFn != nil {
		return m.GetFeedbacksByReviewerIDsFn(ctx, reviewerIDs)
	}
	return nil, nil
}

// validUserRepo returns a mockUserRepository where both reviewer and reviewee exist.
func validUserRepo() *mockUserRepository {
	return &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Test User"}, nil
		},
	}
}

func TestCreateFeedback_Success(t *testing.T) {
	var captured *model.Feedback

	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			captured = feedback
			return feedback, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())

	feedback := &model.Feedback{
		ReviewerID:         "reviewer-1",
		RevieweeID:         "reviewee-1",
		CommunicationScore: 5,
		LeadershipScore:    4,
		TechnicalScore:     5,
		CollaborationScore: 4,
		DeliveryScore:      3,
		StrengthsComment:   "Great communicator",
		WeaknessesComment:  "Could improve delivery",
		Visibility:         "named",
	}

	result, err := svc.CreateFeedback(context.Background(), feedback)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, captured.ID)
	assert.NotEmpty(t, captured.Period)
	assert.False(t, captured.CreatedAt.IsZero())
	assert.False(t, captured.UpdatedAt.IsZero())
	assert.Equal(t, 5, captured.CommunicationScore)
	assert.Equal(t, "Great communicator", captured.StrengthsComment)
	assert.Equal(t, "named", captured.Visibility)
}

func TestCreateFeedback_PeriodCalculation(t *testing.T) {
	tests := []struct {
		month    time.Month
		expected string
	}{
		{time.January, "1-"},
		{time.February, "1-"},
		{time.March, "1-"},
		{time.April, "2-"},
		{time.May, "2-"},
		{time.June, "2-"},
		{time.July, "3-"},
		{time.August, "3-"},
		{time.September, "3-"},
		{time.October, "4-"},
		{time.November, "4-"},
		{time.December, "4-"},
	}

	for _, tt := range tests {
		t.Run(tt.month.String(), func(t *testing.T) {
			quarter := (int(tt.month)-1)/3 + 1
			expectedPrefix := fmt.Sprintf("%d-", quarter)
			assert.Equal(t, tt.expected, expectedPrefix)
		})
	}

	// Verify full period format for current time
	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			return feedback, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	feedback := &model.Feedback{ReviewerID: "r1", RevieweeID: "r2", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	require.NoError(t, err)

	now := time.Now()
	expectedQuarter := (int(now.Month())-1)/3 + 1
	expectedPeriod := fmt.Sprintf("%d-%d", expectedQuarter, now.Year())
	assert.Equal(t, expectedPeriod, result.Period)
}

func TestCreateFeedback_Duplicate(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return &model.Feedback{ID: "existing-feedback"}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())

	feedback := &model.Feedback{
		ReviewerID: "reviewer-1",
		RevieweeID: "reviewee-1",
		Visibility: "named",
	}

	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFeedbackAlreadyExists)
}

func TestCreateFeedback_ReviewerNotFound(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			if id == "reviewer-1" {
				return nil, fmt.Errorf("not found")
			}
			return &model.User{ID: id}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo)

	feedback := &model.Feedback{ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrReviewerNotFound)
}

func TestCreateFeedback_RevieweeNotFound(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			if id == "reviewee-1" {
				return nil, fmt.Errorf("not found")
			}
			return &model.User{ID: id}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo)

	feedback := &model.Feedback{ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrRevieweeNotFound)
}

func TestCreateFeedback_GetFeedbackRepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())

	feedback := &model.Feedback{ReviewerID: "r1", RevieweeID: "r2", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to check existing feedback")
}

func TestGetFeedbacksForUser_Success(t *testing.T) {
	now := time.Now()

	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-1", revieweeID)
			assert.Equal(t, 16, limit)
			assert.Empty(t, cursor)
			return []*model.Feedback{
				{ID: "fb-1", ReviewerID: "reviewer-1", RevieweeID: "user-1", Period: "1-2026", CreatedAt: now, UpdatedAt: now},
				{ID: "fb-2", ReviewerID: "reviewer-2", RevieweeID: "user-1", Period: "1-2026", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetFeedbacksForUser(context.Background(), "user-1", 16, "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "fb-1", result[0].ID)
	assert.Equal(t, "fb-2", result[1].ID)
}

func TestGetFeedbacksForUser_Empty(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return []*model.Feedback{}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetFeedbacksForUser(context.Background(), "user-1", 16, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetFeedbacksForUser_RepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetFeedbacksForUser(context.Background(), "user-1", 16, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get feedbacks")
}

func TestGetGivenFeedbacksForUser_Success(t *testing.T) {
	now := time.Now()

	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDFn: func(_ context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-1", reviewerID)
			assert.Equal(t, 16, limit)
			assert.Empty(t, cursor)
			return []*model.Feedback{
				{ID: "fb-1", ReviewerID: "user-1", RevieweeID: "reviewee-1", Period: "1-2026", CreatedAt: now, UpdatedAt: now},
				{ID: "fb-2", ReviewerID: "user-1", RevieweeID: "reviewee-2", Period: "1-2026", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetGivenFeedbacksForUser(context.Background(), "user-1", 16, "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "fb-1", result[0].ID)
	assert.Equal(t, "fb-2", result[1].ID)
}

func TestGetGivenFeedbacksForUser_Empty(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return []*model.Feedback{}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetGivenFeedbacksForUser(context.Background(), "user-1", 16, "")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetGivenFeedbacksForUser_RepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())
	result, err := svc.GetGivenFeedbacksForUser(context.Background(), "user-1", 16, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get given feedbacks")
}

func TestCreateFeedback_CreateRepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, _ *model.Feedback) (*model.Feedback, error) {
			return nil, fmt.Errorf("firestore write error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo())

	feedback := &model.Feedback{ReviewerID: "r1", RevieweeID: "r2", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to create feedback")
}
