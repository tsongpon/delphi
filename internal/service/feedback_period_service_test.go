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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// activePeriod returns a FeedbackPeriod whose window covers now.
func activePeriod() *model.FeedbackPeriod {
	now := time.Now()
	return &model.FeedbackPeriod{
		ID:        "period-1",
		TeamID:    "team-1",
		Name:      "2026-H1",
		StartDate: now.Add(-24 * time.Hour),
		EndDate:   now.Add(24 * time.Hour),
	}
}

// ---------------------------------------------------------------------------
// CreatePeriod tests
// ---------------------------------------------------------------------------

func TestCreatePeriod_Success(t *testing.T) {
	var stored *model.FeedbackPeriod
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return []*model.FeedbackPeriod{}, nil
		},
		CreatePeriodFn: func(_ context.Context, p *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			stored = p
			return p, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	now := time.Now()
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "2026-H1",
		StartDate: now,
		EndDate:   now.Add(30 * 24 * time.Hour),
		CreatedBy: "manager-1",
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, stored.ID)
	assert.Equal(t, "2026-H1", stored.Name)
	assert.Equal(t, "team-1", stored.TeamID)
	assert.Equal(t, "manager-1", stored.CreatedBy)
	assert.False(t, stored.CreatedAt.IsZero())
	assert.False(t, stored.UpdatedAt.IsZero())
}

func TestCreatePeriod_EndBeforeStart(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{}
	svc := NewFeedbackPeriodService(repo)

	now := time.Now()
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "Bad Period",
		StartDate: now.Add(24 * time.Hour),
		EndDate:   now, // end is before start
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrPeriodEndBeforeStart)
}

func TestCreatePeriod_SameDateStartAndEnd(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{}
	svc := NewFeedbackPeriodService(repo)

	now := time.Now().Truncate(time.Second)
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "Same Day",
		StartDate: now,
		EndDate:   now, // equal, not after
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrPeriodEndBeforeStart)
}

func TestCreatePeriod_NameAlreadyExists(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return []*model.FeedbackPeriod{
				{ID: "existing", Name: "2026-H1", TeamID: "team-1"},
			}, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	now := time.Now()
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "2026-H1", // same name
		StartDate: now,
		EndDate:   now.Add(30 * 24 * time.Hour),
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrPeriodNameExists)
}

func TestCreatePeriod_ListRepoError(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackPeriodService(repo)
	now := time.Now()
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "2026-H1",
		StartDate: now,
		EndDate:   now.Add(30 * 24 * time.Hour),
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to check existing periods")
}

func TestCreatePeriod_CreateRepoError(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return []*model.FeedbackPeriod{}, nil
		},
		CreatePeriodFn: func(_ context.Context, _ *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore write error")
		},
	}

	svc := NewFeedbackPeriodService(repo)
	now := time.Now()
	input := &model.FeedbackPeriod{
		TeamID:    "team-1",
		Name:      "2026-H1",
		StartDate: now,
		EndDate:   now.Add(30 * 24 * time.Hour),
	}

	result, err := svc.CreatePeriod(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// GetActivePeriodForTeam tests
// ---------------------------------------------------------------------------

func TestGetActivePeriodForTeam_Active(t *testing.T) {
	p := activePeriod()
	repo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, teamID string, _ time.Time) (*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return p, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.GetActivePeriodForTeam(context.Background(), "team-1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "period-1", result.ID)
	assert.Equal(t, "2026-H1", result.Name)
}

func TestGetActivePeriodForTeam_None(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.GetActivePeriodForTeam(context.Background(), "team-1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetActivePeriodForTeam_RepoError(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string, _ time.Time) (*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.GetActivePeriodForTeam(context.Background(), "team-1")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// ListPeriodsForTeam tests
// ---------------------------------------------------------------------------

func TestListPeriodsForTeam_Success(t *testing.T) {
	now := time.Now()
	periods := []*model.FeedbackPeriod{
		{ID: "p2", Name: "2026-H2", TeamID: "team-1", StartDate: now.AddDate(0, 6, 0), EndDate: now.AddDate(1, 0, 0)},
		{ID: "p1", Name: "2026-H1", TeamID: "team-1", StartDate: now, EndDate: now.AddDate(0, 6, 0)},
	}

	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, teamID string) ([]*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return periods, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.ListPeriodsForTeam(context.Background(), "team-1")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "p2", result[0].ID)
	assert.Equal(t, "p1", result[1].ID)
}

func TestListPeriodsForTeam_Empty(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return []*model.FeedbackPeriod{}, nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.ListPeriodsForTeam(context.Background(), "team-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListPeriodsForTeam_RepoError(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewFeedbackPeriodService(repo)
	result, err := svc.ListPeriodsForTeam(context.Background(), "team-1")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// DeletePeriod tests
// ---------------------------------------------------------------------------

func TestDeletePeriod_Success(t *testing.T) {
	var deletedTeamID, deletedPeriodID string
	repo := &mockFeedbackPeriodRepository{
		DeletePeriodFn: func(_ context.Context, teamID, periodID string) error {
			deletedTeamID = teamID
			deletedPeriodID = periodID
			return nil
		},
	}

	svc := NewFeedbackPeriodService(repo)
	err := svc.DeletePeriod(context.Background(), "team-1", "period-1")
	require.NoError(t, err)
	assert.Equal(t, "team-1", deletedTeamID)
	assert.Equal(t, "period-1", deletedPeriodID)
}

func TestDeletePeriod_RepoError(t *testing.T) {
	repo := &mockFeedbackPeriodRepository{
		DeletePeriodFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("period not found")
		},
	}

	svc := NewFeedbackPeriodService(repo)
	err := svc.DeletePeriod(context.Background(), "team-1", "nonexistent")
	assert.Error(t, err)
}
