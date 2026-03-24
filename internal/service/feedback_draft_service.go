package service

import (
	"context"
	"fmt"
	"time"

	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

// FeedbackDraftServiceImpl implements handler.FeedbackDraftService.
type FeedbackDraftServiceImpl struct {
	repo       FeedbackDraftRepository
	userRepo   UserRepository
	periodRepo FeedbackPeriodRepository
	feedRepo   FeedbackRepository
}

// NewFeedbackDraftService creates a new FeedbackDraftServiceImpl.
func NewFeedbackDraftService(repo FeedbackDraftRepository, userRepo UserRepository, periodRepo FeedbackPeriodRepository, feedRepo FeedbackRepository) *FeedbackDraftServiceImpl {
	return &FeedbackDraftServiceImpl{repo: repo, userRepo: userRepo, periodRepo: periodRepo, feedRepo: feedRepo}
}

// SaveDraft validates users, resolves the active period, and upserts the draft.
func (s *FeedbackDraftServiceImpl) SaveDraft(ctx context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
	reviewer, err := s.userRepo.GetUserByID(ctx, draft.ReviewerID)
	if err != nil {
		logger.Error("failed to get reviewer", zap.Error(err))
		return nil, ErrReviewerNotFound
	}

	if _, err := s.userRepo.GetUserByID(ctx, draft.RevieweeID); err != nil {
		logger.Error("failed to get reviewee", zap.Error(err))
		return nil, ErrRevieweeNotFound
	}

	now := time.Now()

	activePeriod, err := s.periodRepo.GetActivePeriodForTeam(ctx, reviewer.TeamID, now)
	if err != nil {
		logger.Error("failed to check active period", zap.Error(err))
		return nil, fmt.Errorf("failed to check active period: %w", err)
	}
	if activePeriod == nil {
		return nil, ErrNoActivePeriod
	}
	draft.Period = activePeriod.Name

	// Ensure feedback hasn't already been submitted for this period
	existing, err := s.feedRepo.GetFeedback(ctx, draft.ReviewerID, draft.RevieweeID, draft.Period)
	if err != nil {
		logger.Error("failed to check existing feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing feedback: %w", err)
	}
	if existing != nil {
		return nil, ErrFeedbackAlreadyExists
	}

	now = time.Now()
	draft.UpdatedAt = now
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}

	saved, err := s.repo.UpsertDraft(ctx, draft)
	if err != nil {
		logger.Error("failed to save draft", zap.Error(err))
		return nil, fmt.Errorf("failed to save draft: %w", err)
	}

	return saved, nil
}

// GetDraft returns the draft for the given reviewer/reviewee in the current active period, or nil if none.
func (s *FeedbackDraftServiceImpl) GetDraft(ctx context.Context, reviewerID, revieweeID string) (*model.FeedbackDraft, error) {
	reviewer, err := s.userRepo.GetUserByID(ctx, reviewerID)
	if err != nil {
		logger.Error("failed to get reviewer", zap.Error(err))
		return nil, ErrReviewerNotFound
	}

	activePeriod, err := s.periodRepo.GetActivePeriodForTeam(ctx, reviewer.TeamID, time.Now())
	if err != nil {
		logger.Error("failed to check active period", zap.Error(err))
		return nil, fmt.Errorf("failed to check active period: %w", err)
	}
	if activePeriod == nil {
		return nil, nil
	}

	draft, err := s.repo.GetDraft(ctx, reviewerID, revieweeID, activePeriod.Name)
	if err != nil {
		logger.Error("failed to get draft", zap.Error(err))
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return draft, nil
}

// GetMyDrafts returns all drafts for the reviewer that belong to the current active period.
func (s *FeedbackDraftServiceImpl) GetMyDrafts(ctx context.Context, reviewerID string) ([]*model.FeedbackDraft, error) {
	reviewer, err := s.userRepo.GetUserByID(ctx, reviewerID)
	if err != nil {
		return nil, ErrReviewerNotFound
	}

	activePeriod, err := s.periodRepo.GetActivePeriodForTeam(ctx, reviewer.TeamID, time.Now())
	if err != nil {
		logger.Error("failed to check active period", zap.Error(err))
		return nil, fmt.Errorf("failed to check active period: %w", err)
	}
	if activePeriod == nil {
		return []*model.FeedbackDraft{}, nil
	}

	all, err := s.repo.GetDraftsByReviewerID(ctx, reviewerID)
	if err != nil {
		logger.Error("failed to get drafts", zap.Error(err))
		return nil, fmt.Errorf("failed to get drafts: %w", err)
	}

	// Filter to only the current active period to avoid surfacing stale drafts
	result := make([]*model.FeedbackDraft, 0, len(all))
	for _, d := range all {
		if d.Period == activePeriod.Name {
			result = append(result, d)
		}
	}

	return result, nil
}

// DeleteDraft removes the draft for the given reviewer/reviewee in the current active period.
func (s *FeedbackDraftServiceImpl) DeleteDraft(ctx context.Context, reviewerID, revieweeID string) error {
	reviewer, err := s.userRepo.GetUserByID(ctx, reviewerID)
	if err != nil {
		return ErrReviewerNotFound
	}

	activePeriod, err := s.periodRepo.GetActivePeriodForTeam(ctx, reviewer.TeamID, time.Now())
	if err != nil {
		logger.Error("failed to check active period", zap.Error(err))
		return fmt.Errorf("failed to check active period: %w", err)
	}
	if activePeriod == nil {
		return nil
	}

	err = s.repo.DeleteDraft(ctx, reviewerID, revieweeID, activePeriod.Name)
	if err != nil {
		logger.Error("failed to delete draft", zap.Error(err))
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	return nil
}
