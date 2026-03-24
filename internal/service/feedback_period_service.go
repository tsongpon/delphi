package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

var (
	// ErrPeriodEndBeforeStart is returned when end date is not after start date.
	ErrPeriodEndBeforeStart = errors.New("end date must be after start date")
	// ErrPeriodNameExists is returned when a period with the same name already exists for the team.
	ErrPeriodNameExists = errors.New("a period with this name already exists for the team")
)

// FeedbackPeriodServiceImpl handles feedback period management.
type FeedbackPeriodServiceImpl struct {
	repo FeedbackPeriodRepository
}

// NewFeedbackPeriodService creates a new FeedbackPeriodServiceImpl.
func NewFeedbackPeriodService(repo FeedbackPeriodRepository) *FeedbackPeriodServiceImpl {
	return &FeedbackPeriodServiceImpl{repo: repo}
}

// CreatePeriod validates and persists a new feedback period.
func (s *FeedbackPeriodServiceImpl) CreatePeriod(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
	if !period.EndDate.After(period.StartDate) {
		return nil, ErrPeriodEndBeforeStart
	}

	// Ensure period name is unique within the team to preserve feedback duplicate-check integrity.
	existing, err := s.repo.ListPeriodsForTeam(ctx, period.TeamID)
	if err != nil {
		logger.Error("failed to check existing periods", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing periods: %w", err)
	}
	for _, p := range existing {
		if p.Name == period.Name {
			return nil, ErrPeriodNameExists
		}
	}

	now := time.Now()
	period.ID = uuid.New().String()
	period.CreatedAt = now
	period.UpdatedAt = now

	created, err := s.repo.CreatePeriod(ctx, period)
	if err != nil {
		logger.Error("failed to create period", zap.Error(err))
		return nil, fmt.Errorf("failed to create period: %w", err)
	}

	return created, nil
}

// GetActivePeriodForTeam returns the currently active period for the team, or nil if none.
func (s *FeedbackPeriodServiceImpl) GetActivePeriodForTeam(ctx context.Context, teamID string) (*model.FeedbackPeriod, error) {
	return s.repo.GetActivePeriodForTeam(ctx, teamID, time.Now())
}

// ListPeriodsForTeam returns all periods for the team, newest first.
func (s *FeedbackPeriodServiceImpl) ListPeriodsForTeam(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error) {
	return s.repo.ListPeriodsForTeam(ctx, teamID)
}

// DeletePeriod deletes a period by ID, verifying team ownership.
func (s *FeedbackPeriodServiceImpl) DeletePeriod(ctx context.Context, teamID, periodID string) error {
	return s.repo.DeletePeriod(ctx, teamID, periodID)
}
