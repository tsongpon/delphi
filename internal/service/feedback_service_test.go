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
	CreateFeedbackFn             func(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedbackFn                func(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error)
	GetFeedbacksByRevieweeIDFn   func(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDFn   func(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDsFn  func(ctx context.Context, reviewerIDs []string) ([]*model.Feedback, error)
	GetFeedbacksByRevieweeIDSinceFn func(ctx context.Context, revieweeID string, since time.Time) ([]*model.Feedback, error)
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

func (m *mockFeedbackRepository) GetFeedbacksByRevieweeIDSince(ctx context.Context, revieweeID string, since time.Time) ([]*model.Feedback, error) {
	if m.GetFeedbacksByRevieweeIDSinceFn != nil {
		return m.GetFeedbacksByRevieweeIDSinceFn(ctx, revieweeID, since)
	}
	return nil, nil
}

// mockFeedbackPeriodRepository implements FeedbackPeriodRepository for testing.
type mockFeedbackPeriodRepository struct {
	CreatePeriodFn            func(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error)
	GetActivePeriodForTeamFn  func(ctx context.Context, teamID string, now time.Time) (*model.FeedbackPeriod, error)
	ListPeriodsForTeamFn      func(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error)
	DeletePeriodFn            func(ctx context.Context, teamID, periodID string) error
}

func (m *mockFeedbackPeriodRepository) CreatePeriod(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
	if m.CreatePeriodFn != nil {
		return m.CreatePeriodFn(ctx, period)
	}
	return period, nil
}

func (m *mockFeedbackPeriodRepository) GetActivePeriodForTeam(ctx context.Context, teamID string, now time.Time) (*model.FeedbackPeriod, error) {
	if m.GetActivePeriodForTeamFn != nil {
		return m.GetActivePeriodForTeamFn(ctx, teamID, now)
	}
	return nil, nil
}

func (m *mockFeedbackPeriodRepository) ListPeriodsForTeam(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error) {
	if m.ListPeriodsForTeamFn != nil {
		return m.ListPeriodsForTeamFn(ctx, teamID)
	}
	return []*model.FeedbackPeriod{}, nil
}

func (m *mockFeedbackPeriodRepository) DeletePeriod(ctx context.Context, teamID, periodID string) error {
	if m.DeletePeriodFn != nil {
		return m.DeletePeriodFn(ctx, teamID, periodID)
	}
	return nil
}

// validUserRepo returns a mockUserRepository where reviewer/reviewee exist with a TeamID.
func validUserRepo() *mockUserRepository {
	return &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Test User", TeamID: "team-1"}, nil
		},
	}
}

// validPeriodRepo returns a mockFeedbackPeriodRepository with an active period named "2026-H1".
func validPeriodRepo() *mockFeedbackPeriodRepository {
	return &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return &model.FeedbackPeriod{
				ID:        "period-1",
				TeamID:    "team-1",
				Name:      "2026-H1",
				StartDate: time.Now().Add(-24 * time.Hour),
				EndDate:   time.Now().Add(24 * time.Hour),
			}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// CreateFeedback tests
// ---------------------------------------------------------------------------

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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())

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
	assert.Equal(t, "2026-H1", captured.Period)
	assert.False(t, captured.CreatedAt.IsZero())
	assert.False(t, captured.UpdatedAt.IsZero())
	assert.Equal(t, 5, captured.CommunicationScore)
	assert.Equal(t, "Great communicator", captured.StrengthsComment)
	assert.Equal(t, "named", captured.Visibility)
}

func TestCreateFeedback_UsesActivePeriodName(t *testing.T) {
	const periodName = "2026-Annual"

	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, period string) (*model.Feedback, error) {
			assert.Equal(t, periodName, period)
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			return feedback, nil
		},
	}

	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, teamID string, _ time.Time) (*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return &model.FeedbackPeriod{ID: "p1", Name: periodName}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), periodRepo)
	result, err := svc.CreateFeedback(context.Background(), &model.Feedback{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named",
	})
	require.NoError(t, err)
	assert.Equal(t, periodName, result.Period)
}

func TestCreateFeedback_NoActivePeriod(t *testing.T) {
	repo := &mockFeedbackRepository{}
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil // no active period
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), periodRepo)
	result, err := svc.CreateFeedback(context.Background(), &model.Feedback{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrNoActivePeriod)
}

func TestCreateFeedback_PeriodRepoError(t *testing.T) {
	repo := &mockFeedbackRepository{}
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), periodRepo)
	result, err := svc.CreateFeedback(context.Background(), &model.Feedback{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to check active period")
}

func TestCreateFeedback_Duplicate(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return &model.Feedback{ID: "existing-feedback"}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())

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
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())

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
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())

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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())

	feedback := &model.Feedback{ReviewerID: "r1", RevieweeID: "r2", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to check existing feedback")
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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())

	feedback := &model.Feedback{ReviewerID: "r1", RevieweeID: "r2", Visibility: "named"}
	result, err := svc.CreateFeedback(context.Background(), feedback)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to create feedback")
}

// ---------------------------------------------------------------------------
// GetFeedbacksForUser tests
// ---------------------------------------------------------------------------

func TestGetFeedbacksForUser_Success(t *testing.T) {
	now := time.Now()

	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-1", revieweeID)
			assert.Equal(t, 16, limit)
			assert.Empty(t, cursor)
			return []*model.Feedback{
				{ID: "fb-1", ReviewerID: "reviewer-1", RevieweeID: "user-1", Period: "2026-H1", CreatedAt: now, UpdatedAt: now},
				{ID: "fb-2", ReviewerID: "reviewer-2", RevieweeID: "user-1", Period: "2026-H1", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
	result, err := svc.GetFeedbacksForUser(context.Background(), "user-1", 16, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get feedbacks")
}

// ---------------------------------------------------------------------------
// GetGivenFeedbacksForUser tests
// ---------------------------------------------------------------------------

func TestGetGivenFeedbacksForUser_Success(t *testing.T) {
	now := time.Now()

	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDFn: func(_ context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-1", reviewerID)
			assert.Equal(t, 16, limit)
			assert.Empty(t, cursor)
			return []*model.Feedback{
				{ID: "fb-1", ReviewerID: "user-1", RevieweeID: "reviewee-1", Period: "2026-H1", CreatedAt: now, UpdatedAt: now},
				{ID: "fb-2", ReviewerID: "user-1", RevieweeID: "reviewee-2", Period: "2026-H1", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
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

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
	result, err := svc.GetGivenFeedbacksForUser(context.Background(), "user-1", 16, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get given feedbacks")
}

// ---------------------------------------------------------------------------
// GetTeamFeedbacks tests
// ---------------------------------------------------------------------------

func TestGetTeamFeedbacks_Success(t *testing.T) {
	now := time.Now()
	members := []*model.User{
		{ID: "member-1", TeamID: "team-1"},
		{ID: "member-2", TeamID: "team-1"},
	}

	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDsFn: func(_ context.Context, reviewerIDs []string) ([]*model.Feedback, error) {
			return []*model.Feedback{
				{ID: "fb-1", ReviewerID: "member-1", RevieweeID: "member-2", Period: "2026-H1", CreatedAt: now},
				{ID: "fb-2", ReviewerID: "member-2", RevieweeID: "member-1", Period: "2026-H1", CreatedAt: now},
				// external reviewee — should be filtered out
				{ID: "fb-3", ReviewerID: "member-1", RevieweeID: "external-user", Period: "2026-H1", CreatedAt: now},
			}, nil
		},
	}

	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, teamID string) ([]*model.User, error) {
			assert.Equal(t, "team-1", teamID)
			return members, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamFeedbacks(context.Background(), "team-1")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "fb-1", result[0].ID)
	assert.Equal(t, "fb-2", result[1].ID)
}

func TestGetTeamFeedbacks_EmptyTeam(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return []*model.User{}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamFeedbacks(context.Background(), "team-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetTeamFeedbacks_GetMembersError(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamFeedbacks(context.Background(), "team-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get team members")
}

func TestGetTeamFeedbacks_GetFeedbacksError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDsFn: func(_ context.Context, _ []string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}
	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return []*model.User{{ID: "member-1"}}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamFeedbacks(context.Background(), "team-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get team feedbacks")
}

// ---------------------------------------------------------------------------
// GetTeamDashboard tests
// ---------------------------------------------------------------------------

func TestGetTeamDashboard_Success(t *testing.T) {
	now := time.Now()
	members := []*model.User{
		{ID: "member-1", Name: "Alice", Title: "Engineer", Email: "alice@example.com", TeamID: "team-1"},
		{ID: "member-2", Name: "Bob", Title: "Manager", Email: "bob@example.com", TeamID: "team-1"},
	}

	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDsFn: func(_ context.Context, _ []string) ([]*model.Feedback, error) {
			return []*model.Feedback{
				{
					ID: "fb-1", ReviewerID: "member-2", RevieweeID: "member-1",
					CommunicationScore: 5, LeadershipScore: 4, TechnicalScore: 5,
					CollaborationScore: 4, DeliveryScore: 3, CreatedAt: now,
				},
				{
					ID: "fb-2", ReviewerID: "member-1", RevieweeID: "member-2",
					CommunicationScore: 4, LeadershipScore: 3, TechnicalScore: 4,
					CollaborationScore: 5, DeliveryScore: 4, CreatedAt: now,
				},
			}, nil
		},
	}

	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return members, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamDashboard(context.Background(), "team-1")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.TeamMembers)
	assert.Equal(t, 2, result.TotalFeedbacks)
	assert.Equal(t, 100, result.FeedbackCoverage)
	assert.Greater(t, result.AvgTeamScore, 0.0)
	assert.Len(t, result.Members, 2)
}

func TestGetTeamDashboard_NoFeedbacks(t *testing.T) {
	members := []*model.User{
		{ID: "member-1", Name: "Alice", TeamID: "team-1"},
	}

	repo := &mockFeedbackRepository{
		GetFeedbacksByReviewerIDsFn: func(_ context.Context, _ []string) ([]*model.Feedback, error) {
			return []*model.Feedback{}, nil
		},
	}

	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return members, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamDashboard(context.Background(), "team-1")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TeamMembers)
	assert.Equal(t, 0, result.TotalFeedbacks)
	assert.Equal(t, 0, result.FeedbackCoverage)
	assert.Equal(t, 0.0, result.AvgTeamScore)
}

func TestGetTeamDashboard_GetMembersError(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetTeamDashboard(context.Background(), "team-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get team members")
}

// ---------------------------------------------------------------------------
// ExportFeedbacksForUser tests
// ---------------------------------------------------------------------------

func TestExportFeedbacksForUser_Success_Named(t *testing.T) {
	now := time.Now()
	recent := now.Add(-24 * time.Hour)
	old := now.AddDate(-2, 0, 0)

	feedbacks := []*model.Feedback{
		{
			ID: "fb-1", ReviewerID: "reviewer-1", RevieweeID: "user-1",
			Visibility: "named", CreatedAt: recent,
		},
		{
			ID: "fb-2", ReviewerID: "reviewer-2", RevieweeID: "user-1",
			Visibility: "anonymous", CreatedAt: recent,
		},
		// older than 12 months — should be filtered out
		{
			ID: "fb-old", ReviewerID: "reviewer-1", RevieweeID: "user-1",
			Visibility: "named", CreatedAt: old,
		},
	}

	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-1", revieweeID)
			assert.Equal(t, 1000, limit)
			assert.Empty(t, cursor)
			return feedbacks, nil
		},
	}

	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Reviewer " + id, TeamID: "team-1"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.ExportFeedbacksForUser(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// named feedback should have reviewer name
	assert.Equal(t, "fb-1", result[0].Feedback.ID)
	assert.NotEmpty(t, result[0].ReviewerName)

	// anonymous feedback should have empty reviewer name
	assert.Equal(t, "fb-2", result[1].Feedback.ID)
	assert.Empty(t, result[1].ReviewerName)
}

func TestExportFeedbacksForUser_Empty(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return []*model.Feedback{}, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
	result, err := svc.ExportFeedbacksForUser(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestExportFeedbacksForUser_RepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
	result, err := svc.ExportFeedbacksForUser(context.Background(), "user-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get feedbacks for export")
}

func TestExportFeedbacksForUser_FiltersOldFeedbacks(t *testing.T) {
	now := time.Now()
	feedbacks := []*model.Feedback{
		{ID: "recent", ReviewerID: "r1", Visibility: "named", CreatedAt: now.Add(-30 * 24 * time.Hour)},
		{ID: "old", ReviewerID: "r1", Visibility: "named", CreatedAt: now.AddDate(-2, 0, 0)},
	}

	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return feedbacks, nil
		},
	}

	svc := NewFeedbackService(repo, validUserRepo(), validPeriodRepo())
	result, err := svc.ExportFeedbacksForUser(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "recent", result[0].Feedback.ID)
}

// ---------------------------------------------------------------------------
// GetFeedbacksForMember tests
// ---------------------------------------------------------------------------

func TestGetFeedbacksForMember_Success(t *testing.T) {
	now := time.Now()

	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, memberID string, limit int, cursor string) ([]*model.Feedback, error) {
			assert.Equal(t, "member-1", memberID)
			assert.Equal(t, 10, limit)
			return []*model.Feedback{
				{ID: "fb-1", RevieweeID: "member-1", CreatedAt: now},
			}, nil
		},
	}

	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetFeedbacksForMember(context.Background(), "team-1", "member-1", 10, "")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "fb-1", result[0].ID)
}

func TestGetFeedbacksForMember_MemberNotFound(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetFeedbacksForMember(context.Background(), "team-1", "member-1", 10, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrRevieweeNotFound)
}

func TestGetFeedbacksForMember_MemberNotInTeam(t *testing.T) {
	repo := &mockFeedbackRepository{}
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, TeamID: "other-team"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetFeedbacksForMember(context.Background(), "team-1", "member-1", 10, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrMemberNotInTeam)
}

func TestGetFeedbacksForMember_RepoError(t *testing.T) {
	repo := &mockFeedbackRepository{
		GetFeedbacksByRevieweeIDFn: func(_ context.Context, _ string, _ int, _ string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := NewFeedbackService(repo, userRepo, validPeriodRepo())
	result, err := svc.GetFeedbacksForMember(context.Background(), "team-1", "member-1", 10, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get member feedbacks")
}

// ---------------------------------------------------------------------------
// round2 helper test
// ---------------------------------------------------------------------------

func TestRound2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{3.14159, 3.14},
		{3.145, 3.15},
		{3.0, 3.0},
		{0.0, 0.0},
		{4.666, 4.67},
	}

	for _, tt := range tests {
		result := round2(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}
