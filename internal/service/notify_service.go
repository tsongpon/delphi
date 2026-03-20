package service

import (
	"context"
	"time"

	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

// NotifyResult summarises the outcome of a feedback digest run.
type NotifyResult struct {
	Notified int
	Skipped  int
}

// NotifyService sends daily feedback digest emails to users who received new feedback yesterday.
type NotifyService struct {
	userRepo     UserRepository
	feedbackRepo FeedbackRepository
	emailSender  EmailSender
}

// NewNotifyService creates a new NotifyService.
func NewNotifyService(userRepo UserRepository, feedbackRepo FeedbackRepository, emailSender EmailSender) *NotifyService {
	return &NotifyService{
		userRepo:     userRepo,
		feedbackRepo: feedbackRepo,
		emailSender:  emailSender,
	}
}

// SendFeedbackDigest iterates users, checks for feedbacks received since yesterday (UTC),
// and sends a digest email to each user who has at least one new feedback.
// When teamID is non-empty only members of that team are notified.
func (s *NotifyService) SendFeedbackDigest(ctx context.Context, teamID string) (*NotifyResult, error) {
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	since := startOfToday.AddDate(0, 0, -1) // start of yesterday

	var users []*model.User
	var err error
	if teamID != "" {
		users, err = s.userRepo.GetUsersByTeamID(ctx, teamID)
	} else {
		users, err = s.userRepo.GetAllUsers(ctx)
	}
	if err != nil {
		return nil, err
	}

	result := &NotifyResult{}

	for _, u := range users {
		feedbacks, err := s.feedbackRepo.GetFeedbacksByRevieweeIDSince(ctx, u.ID, since)
		if err != nil {
			logger.Error("failed to get feedbacks for user during digest", zap.String("user_id", u.ID), zap.Error(err))
			result.Skipped++
			continue
		}

		if len(feedbacks) == 0 {
			result.Skipped++
			continue
		}

		if err := s.emailSender.SendFeedbackDigest(ctx, u.Name, u.Email, len(feedbacks)); err != nil {
			logger.Error("failed to send digest email", zap.String("user_id", u.ID), zap.String("email", u.Email), zap.Error(err))
			result.Skipped++
			continue
		}

		result.Notified++
	}

	return result, nil
}
