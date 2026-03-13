package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

var (
	// ErrFeedbackAlreadyExists is returned when a feedback already exists for the same reviewer, reviewee, and period.
	ErrFeedbackAlreadyExists = errors.New("feedback already exists for this period")
	// ErrReviewerNotFound is returned when the reviewer user does not exist.
	ErrReviewerNotFound = errors.New("reviewer not found")
	// ErrRevieweeNotFound is returned when the reviewee user does not exist.
	ErrRevieweeNotFound = errors.New("reviewee not found")
	// ErrMemberNotInTeam is returned when the member does not belong to the given team.
	ErrMemberNotInTeam = errors.New("member does not belong to this team")
)

// TeamDashboard holds aggregated team performance data.
type TeamDashboard struct {
	TeamMembers      int
	AvgTeamScore     float64
	TotalFeedbacks   int
	FeedbackCoverage int
	Members          []MemberDashboard
}

// MemberDashboard holds a single member's aggregated feedback scores.
type MemberDashboard struct {
	ID            string
	Name          string
	Title         string
	Email         string
	AvgScore      float64
	FeedbackCount int
	Communication float64
	Leadership    float64
	Technical     float64
	Collaboration float64
	Delivery      float64
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

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

// GetFeedbacksForUser returns feedbacks where the given user is the reviewee, with cursor pagination.
func (s *FeedbackServiceImpl) GetFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error) {
	feedbacks, err := s.repo.GetFeedbacksByRevieweeID(ctx, userID, limit, cursor)
	if err != nil {
		logger.Error("failed to get feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to get feedbacks: %w", err)
	}
	return feedbacks, nil
}

// GetTeamFeedbacks returns all feedbacks where both reviewer and reviewee are members of the given team.
func (s *FeedbackServiceImpl) GetTeamFeedbacks(ctx context.Context, teamID string) ([]*model.Feedback, error) {
	members, err := s.userRepo.GetUsersByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	if len(members) == 0 {
		return []*model.Feedback{}, nil
	}

	memberIDs := make([]string, len(members))
	memberIDSet := make(map[string]struct{}, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
		memberIDSet[m.ID] = struct{}{}
	}

	feedbacks, err := s.repo.GetFeedbacksByReviewerIDs(ctx, memberIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get team feedbacks: %w", err)
	}

	result := make([]*model.Feedback, 0)
	for _, f := range feedbacks {
		if _, ok := memberIDSet[f.RevieweeID]; ok {
			result = append(result, f)
		}
	}
	return result, nil
}

// GetGivenFeedbacksForUser returns feedbacks where the given user is the reviewer, with cursor pagination.
func (s *FeedbackServiceImpl) GetGivenFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error) {
	feedbacks, err := s.repo.GetFeedbacksByReviewerID(ctx, userID, limit, cursor)
	if err != nil {
		logger.Error("failed to get given feedbacks", zap.Error(err))
		return nil, fmt.Errorf("failed to get given feedbacks: %w", err)
	}
	return feedbacks, nil
}

// GetTeamDashboard returns aggregated scores and stats for all members of the given team.
func (s *FeedbackServiceImpl) GetTeamDashboard(ctx context.Context, teamID string) (*TeamDashboard, error) {
	members, err := s.userRepo.GetUsersByTeamID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	feedbacks, err := s.GetTeamFeedbacks(ctx, teamID)
	if err != nil {
		return nil, err
	}

	type raw struct {
		communication, leadership, technical, collaboration, delivery int
		count                                                         int
	}
	scoresByMember := make(map[string]*raw, len(members))
	for _, m := range members {
		scoresByMember[m.ID] = &raw{}
	}
	for _, f := range feedbacks {
		if r, ok := scoresByMember[f.RevieweeID]; ok {
			r.communication += f.CommunicationScore
			r.leadership += f.LeadershipScore
			r.technical += f.TechnicalScore
			r.collaboration += f.CollaborationScore
			r.delivery += f.DeliveryScore
			r.count++
		}
	}

	memberDashboards := make([]MemberDashboard, 0, len(members))
	var totalAvgScore float64
	membersWithFeedback := 0
	for _, m := range members {
		r := scoresByMember[m.ID]
		md := MemberDashboard{
			ID:            m.ID,
			Name:          m.Name,
			Title:         m.Title,
			Email:         m.Email,
			FeedbackCount: r.count,
		}
		if r.count > 0 {
			md.Communication = round2(float64(r.communication) / float64(r.count))
			md.Leadership = round2(float64(r.leadership) / float64(r.count))
			md.Technical = round2(float64(r.technical) / float64(r.count))
			md.Collaboration = round2(float64(r.collaboration) / float64(r.count))
			md.Delivery = round2(float64(r.delivery) / float64(r.count))
			md.AvgScore = round2((md.Communication + md.Leadership + md.Technical + md.Collaboration + md.Delivery) / 5)
			totalAvgScore += md.AvgScore
			membersWithFeedback++
		}
		memberDashboards = append(memberDashboards, md)
	}

	coverage := 0
	if len(members) > 0 {
		coverage = membersWithFeedback * 100 / len(members)
	}
	avgTeamScore := 0.0
	if membersWithFeedback > 0 {
		avgTeamScore = round2(totalAvgScore / float64(membersWithFeedback))
	}

	return &TeamDashboard{
		TeamMembers:      len(members),
		AvgTeamScore:     avgTeamScore,
		TotalFeedbacks:   len(feedbacks),
		FeedbackCoverage: coverage,
		Members:          memberDashboards,
	}, nil
}

// GetFeedbacksForMember returns paginated feedbacks for a specific member, verifying they belong to the team.
func (s *FeedbackServiceImpl) GetFeedbacksForMember(ctx context.Context, teamID, memberID string, limit int, cursor string) ([]*model.Feedback, error) {
	member, err := s.userRepo.GetUserByID(ctx, memberID)
	if err != nil {
		return nil, ErrRevieweeNotFound
	}
	if member.TeamID != teamID {
		return nil, ErrMemberNotInTeam
	}
	feedbacks, err := s.repo.GetFeedbacksByRevieweeID(ctx, memberID, limit, cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to get member feedbacks: %w", err)
	}
	return feedbacks, nil
}
