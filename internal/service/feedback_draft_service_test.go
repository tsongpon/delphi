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

// mockFeedbackDraftRepository implements FeedbackDraftRepository for testing.
type mockFeedbackDraftRepository struct {
	UpsertDraftFn           func(ctx context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error)
	GetDraftFn              func(ctx context.Context, reviewerID, revieweeID, period string) (*model.FeedbackDraft, error)
	GetDraftsByReviewerIDFn func(ctx context.Context, reviewerID string) ([]*model.FeedbackDraft, error)
	DeleteDraftFn           func(ctx context.Context, reviewerID, revieweeID, period string) error
}

func (m *mockFeedbackDraftRepository) UpsertDraft(ctx context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
	if m.UpsertDraftFn != nil {
		return m.UpsertDraftFn(ctx, draft)
	}
	return draft, nil
}

func (m *mockFeedbackDraftRepository) GetDraft(ctx context.Context, reviewerID, revieweeID, period string) (*model.FeedbackDraft, error) {
	if m.GetDraftFn != nil {
		return m.GetDraftFn(ctx, reviewerID, revieweeID, period)
	}
	return nil, nil
}

func (m *mockFeedbackDraftRepository) GetDraftsByReviewerID(ctx context.Context, reviewerID string) ([]*model.FeedbackDraft, error) {
	if m.GetDraftsByReviewerIDFn != nil {
		return m.GetDraftsByReviewerIDFn(ctx, reviewerID)
	}
	return nil, nil
}

func (m *mockFeedbackDraftRepository) DeleteDraft(ctx context.Context, reviewerID, revieweeID, period string) error {
	if m.DeleteDraftFn != nil {
		return m.DeleteDraftFn(ctx, reviewerID, revieweeID, period)
	}
	return nil
}

// newTestDraftService is a helper that creates a FeedbackDraftServiceImpl with the given mocks.
func newTestDraftService(
	draftRepo *mockFeedbackDraftRepository,
	userRepo *mockUserRepository,
	periodRepo *mockFeedbackPeriodRepository,
	feedRepo *mockFeedbackRepository,
) *FeedbackDraftServiceImpl {
	return NewFeedbackDraftService(draftRepo, userRepo, periodRepo, feedRepo)
}

// noExistingFeedback returns a mockFeedbackRepository where GetFeedback always returns nil (no duplicate).
func noExistingFeedback() *mockFeedbackRepository {
	return &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
	}
}

// ---------------------------------------------------------------------------
// SaveDraft tests
// ---------------------------------------------------------------------------

func TestSaveDraft_Success(t *testing.T) {
	var upserted *model.FeedbackDraft

	draftRepo := &mockFeedbackDraftRepository{
		UpsertDraftFn: func(_ context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			upserted = draft
			return draft, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())

	draft := &model.FeedbackDraft{
		ReviewerID:         "reviewer-1",
		RevieweeID:         "reviewee-1",
		CommunicationScore: 4,
		LeadershipScore:    3,
		StrengthsComment:   "Good work",
		Visibility:         "named",
	}

	result, err := svc.SaveDraft(context.Background(), draft)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "2026-H1", upserted.Period)
	assert.False(t, upserted.UpdatedAt.IsZero())
	assert.False(t, upserted.CreatedAt.IsZero())
	assert.Equal(t, 4, upserted.CommunicationScore)
	assert.Equal(t, "Good work", upserted.StrengthsComment)
}

func TestSaveDraft_SetsActivePeriod(t *testing.T) {
	const periodName = "2026-Q2"

	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, teamID string, _ time.Time) (*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return &model.FeedbackPeriod{Name: periodName}, nil
		},
	}

	var capturedPeriod string
	draftRepo := &mockFeedbackDraftRepository{
		UpsertDraftFn: func(_ context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			capturedPeriod = draft.Period
			return draft, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), periodRepo, noExistingFeedback())
	_, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1", Visibility: "named",
	})
	require.NoError(t, err)
	assert.Equal(t, periodName, capturedPeriod)
}

func TestSaveDraft_ReviewerNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			if id == "reviewer-1" {
				return nil, fmt.Errorf("not found")
			}
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, userRepo, validPeriodRepo(), noExistingFeedback())
	result, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrReviewerNotFound)
}

func TestSaveDraft_RevieweeNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			if id == "reviewee-1" {
				return nil, fmt.Errorf("not found")
			}
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, userRepo, validPeriodRepo(), noExistingFeedback())
	result, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrRevieweeNotFound)
}

func TestSaveDraft_NoActivePeriod(t *testing.T) {
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, validUserRepo(), periodRepo, noExistingFeedback())
	result, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrNoActivePeriod)
}

func TestSaveDraft_FeedbackAlreadySubmitted(t *testing.T) {
	feedRepo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return &model.Feedback{ID: "existing-feedback"}, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, validUserRepo(), validPeriodRepo(), feedRepo)
	result, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrFeedbackAlreadyExists)
}

func TestSaveDraft_UpsertRepoError(t *testing.T) {
	draftRepo := &mockFeedbackDraftRepository{
		UpsertDraftFn: func(_ context.Context, _ *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	result, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1", RevieweeID: "reviewee-1",
	})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to save draft")
}

func TestSaveDraft_PreservesExistingCreatedAt(t *testing.T) {
	existingCreatedAt := time.Now().Add(-time.Hour)
	var capturedCreatedAt time.Time

	draftRepo := &mockFeedbackDraftRepository{
		UpsertDraftFn: func(_ context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			capturedCreatedAt = draft.CreatedAt
			return draft, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	_, err := svc.SaveDraft(context.Background(), &model.FeedbackDraft{
		ReviewerID: "reviewer-1",
		RevieweeID: "reviewee-1",
		CreatedAt:  existingCreatedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, existingCreatedAt, capturedCreatedAt)
}

// ---------------------------------------------------------------------------
// GetDraft tests
// ---------------------------------------------------------------------------

func TestGetDraft_ReturnsDraftForActivePeriod(t *testing.T) {
	now := time.Now()
	expected := &model.FeedbackDraft{
		ID:         "draft-1",
		ReviewerID: "reviewer-1",
		RevieweeID: "reviewee-1",
		Period:     "2026-H1",
		UpdatedAt:  now,
	}

	draftRepo := &mockFeedbackDraftRepository{
		GetDraftFn: func(_ context.Context, reviewerID, revieweeID, period string) (*model.FeedbackDraft, error) {
			assert.Equal(t, "reviewer-1", reviewerID)
			assert.Equal(t, "reviewee-1", revieweeID)
			assert.Equal(t, "2026-H1", period)
			return expected, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetDraft(context.Background(), "reviewer-1", "reviewee-1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "draft-1", result.ID)
}

func TestGetDraft_ReturnsNilWhenNoDraft(t *testing.T) {
	draftRepo := &mockFeedbackDraftRepository{
		GetDraftFn: func(_ context.Context, _, _, _ string) (*model.FeedbackDraft, error) {
			return nil, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetDraft(context.Background(), "reviewer-1", "reviewee-1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetDraft_ReturnsNilWhenNoActivePeriod(t *testing.T) {
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, validUserRepo(), periodRepo, noExistingFeedback())
	result, err := svc.GetDraft(context.Background(), "reviewer-1", "reviewee-1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetDraft_ReviewerNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, userRepo, validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetDraft(context.Background(), "reviewer-1", "reviewee-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrReviewerNotFound)
}

// ---------------------------------------------------------------------------
// GetMyDrafts tests
// ---------------------------------------------------------------------------

func TestGetMyDrafts_FiltersToActivePeriod(t *testing.T) {
	activePeriodName := "2026-H1"
	allDrafts := []*model.FeedbackDraft{
		{ID: "draft-current", RevieweeID: "r1", Period: activePeriodName},
		{ID: "draft-old", RevieweeID: "r2", Period: "2025-H2"},
	}

	draftRepo := &mockFeedbackDraftRepository{
		GetDraftsByReviewerIDFn: func(_ context.Context, reviewerID string) ([]*model.FeedbackDraft, error) {
			assert.Equal(t, "reviewer-1", reviewerID)
			return allDrafts, nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetMyDrafts(context.Background(), "reviewer-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "draft-current", result[0].ID)
}

func TestGetMyDrafts_ReturnsEmptyWhenNoActivePeriod(t *testing.T) {
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, validUserRepo(), periodRepo, noExistingFeedback())
	result, err := svc.GetMyDrafts(context.Background(), "reviewer-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetMyDrafts_ReviewerNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, userRepo, validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetMyDrafts(context.Background(), "reviewer-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrReviewerNotFound)
}

func TestGetMyDrafts_RepoError(t *testing.T) {
	draftRepo := &mockFeedbackDraftRepository{
		GetDraftsByReviewerIDFn: func(_ context.Context, _ string) ([]*model.FeedbackDraft, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	result, err := svc.GetMyDrafts(context.Background(), "reviewer-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get drafts")
}

// ---------------------------------------------------------------------------
// DeleteDraft tests
// ---------------------------------------------------------------------------

func TestDeleteDraft_Success(t *testing.T) {
	var deletedReviewerID, deletedRevieweeID, deletedPeriod string

	draftRepo := &mockFeedbackDraftRepository{
		DeleteDraftFn: func(_ context.Context, reviewerID, revieweeID, period string) error {
			deletedReviewerID = reviewerID
			deletedRevieweeID = revieweeID
			deletedPeriod = period
			return nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), validPeriodRepo(), noExistingFeedback())
	err := svc.DeleteDraft(context.Background(), "reviewer-1", "reviewee-1")
	require.NoError(t, err)
	assert.Equal(t, "reviewer-1", deletedReviewerID)
	assert.Equal(t, "reviewee-1", deletedRevieweeID)
	assert.Equal(t, "2026-H1", deletedPeriod)
}

func TestDeleteDraft_NoActivePeriod_NoError(t *testing.T) {
	periodRepo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	deleteCalled := false
	draftRepo := &mockFeedbackDraftRepository{
		DeleteDraftFn: func(_ context.Context, _, _, _ string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := newTestDraftService(draftRepo, validUserRepo(), periodRepo, noExistingFeedback())
	err := svc.DeleteDraft(context.Background(), "reviewer-1", "reviewee-1")
	require.NoError(t, err)
	assert.False(t, deleteCalled, "delete should not be called when no active period")
}

func TestDeleteDraft_ReviewerNotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := newTestDraftService(&mockFeedbackDraftRepository{}, userRepo, validPeriodRepo(), noExistingFeedback())
	err := svc.DeleteDraft(context.Background(), "reviewer-1", "reviewee-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrReviewerNotFound)
}

// ---------------------------------------------------------------------------
// CreateFeedback draft cleanup tests
// ---------------------------------------------------------------------------

func TestCreateFeedback_DeletesDraftOnSuccess(t *testing.T) {
	var deletedReviewerID, deletedRevieweeID, deletedPeriod string

	feedRepo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			return feedback, nil
		},
	}

	draftRepo := &mockFeedbackDraftRepository{
		DeleteDraftFn: func(_ context.Context, reviewerID, revieweeID, period string) error {
			deletedReviewerID = reviewerID
			deletedRevieweeID = revieweeID
			deletedPeriod = period
			return nil
		},
	}

	svc := NewFeedbackService(feedRepo, validUserRepo(), validPeriodRepo(), draftRepo)
	_, err := svc.CreateFeedback(context.Background(), &model.Feedback{
		ReviewerID: "reviewer-1",
		RevieweeID: "reviewee-1",
		Visibility: "named",
	})
	require.NoError(t, err)
	assert.Equal(t, "reviewer-1", deletedReviewerID)
	assert.Equal(t, "reviewee-1", deletedRevieweeID)
	assert.Equal(t, "2026-H1", deletedPeriod)
}

func TestCreateFeedback_ContinuesIfDraftDeleteFails(t *testing.T) {
	feedRepo := &mockFeedbackRepository{
		GetFeedbackFn: func(_ context.Context, _, _, _ string) (*model.Feedback, error) {
			return nil, nil
		},
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			return feedback, nil
		},
	}

	draftRepo := &mockFeedbackDraftRepository{
		DeleteDraftFn: func(_ context.Context, _, _, _ string) error {
			return fmt.Errorf("firestore error deleting draft")
		},
	}

	svc := NewFeedbackService(feedRepo, validUserRepo(), validPeriodRepo(), draftRepo)
	result, err := svc.CreateFeedback(context.Background(), &model.Feedback{
		ReviewerID: "reviewer-1",
		RevieweeID: "reviewee-1",
		Visibility: "named",
	})
	// CreateFeedback should succeed even if draft delete fails
	require.NoError(t, err)
	assert.NotNil(t, result)
}
