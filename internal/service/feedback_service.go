package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/model"
)

var (
	// ErrFeedbackAlreadyExists is returned when a feedback already exists for the same reviewer, reviewee, and period.
	ErrFeedbackAlreadyExists = errors.New("feedback already exists for this period")
	// ErrReviewerNotFound is returned when the reviewer user does not exist.
	ErrReviewerNotFound = errors.New("reviewer not found")
	// ErrRevieweeNotFound is returned when the reviewee user does not exist.
	ErrRevieweeNotFound = errors.New("reviewee not found")
)

// FeedbackServiceImpl implements handler.FeedbackService.
type FeedbackServiceImpl struct {
	repo     FeedbackRepository
	userRepo UserRepository
}

// NewFeedbackService creates a new FeedbackServiceImpl.
func NewFeedbackService(repo FeedbackRepository, userRepo UserRepository) *FeedbackServiceImpl {
	return &FeedbackServiceImpl{repo: repo, userRepo: userRepo}
}

// CreateFeedback validates users, calculates the period, checks for duplicates, and persists the feedback.
func (s *FeedbackServiceImpl) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	// Validate reviewer exists
	if _, err := s.userRepo.GetUserByID(ctx, feedback.ReviewerID); err != nil {
		return nil, ErrReviewerNotFound
	}

	// Validate reviewee exists
	if _, err := s.userRepo.GetUserByID(ctx, feedback.RevieweeID); err != nil {
		return nil, ErrRevieweeNotFound
	}

	now := time.Now()

	// Calculate period: quarter-year (e.g. "1-2026" for Q1 2026)
	quarter := (int(now.Month())-1)/3 + 1
	feedback.Period = fmt.Sprintf("%d-%d", quarter, now.Year())

	// Check for duplicate
	existing, err := s.repo.GetFeedback(ctx, feedback.ReviewerID, feedback.RevieweeID, feedback.Period)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing feedback: %w", err)
	}
	if existing != nil {
		return nil, ErrFeedbackAlreadyExists
	}

	feedback.ID = uuid.New().String()
	feedback.CreatedAt = now
	feedback.UpdatedAt = now

	created, err := s.repo.CreateFeedback(ctx, feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	return created, nil
}
