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
	// ErrNoActivePeriod is returned when there is no active feedback period for the team.
	ErrNoActivePeriod = errors.New("no active feedback period")
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
	repo       FeedbackRepository
	userRepo   UserRepository
	periodRepo FeedbackPeriodRepository
	draftRepo  FeedbackDraftRepository
}

// NewFeedbackService creates a new FeedbackServiceImpl.
func NewFeedbackService(repo FeedbackRepository, userRepo UserRepository, periodRepo FeedbackPeriodRepository, draftRepo FeedbackDraftRepository) *FeedbackServiceImpl {
	return &FeedbackServiceImpl{repo: repo, userRepo: userRepo, periodRepo: periodRepo, draftRepo: draftRepo}
}

// CreateFeedback validates users, looks up the active period, checks for duplicates, and persists the feedback.
func (s *FeedbackServiceImpl) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	// Validate reviewer exists and get team ID for period lookup
	reviewer, err := s.userRepo.GetUserByID(ctx, feedback.ReviewerID)
	if err != nil {
		return nil, ErrReviewerNotFound
	}

	// Validate reviewee exists
	if _, err := s.userRepo.GetUserByID(ctx, feedback.RevieweeID); err != nil {
		return nil, ErrRevieweeNotFound
	}

	now := time.Now()

	// Look up the active period for the reviewer's team
	activePeriod, err := s.periodRepo.GetActivePeriodForTeam(ctx, reviewer.TeamID, now)
	if err != nil {
		logger.Error("failed to check active period", zap.Error(err))
		return nil, fmt.Errorf("failed to check active period: %w", err)
	}
	if activePeriod == nil {
		return nil, ErrNoActivePeriod
	}
	feedback.Period = activePeriod.Name

	// Check for duplicate
	existing, err := s.repo.GetFeedback(ctx, feedback.ReviewerID, feedback.RevieweeID, feedback.Period)
	if err != nil {
		logger.Error("failed to check existing feedback", zap.Error(err))
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
		logger.Error("failed to create feedback", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	// Best-effort: delete any saved draft for this reviewer/reviewee/period
	if s.draftRepo != nil {
		if delErr := s.draftRepo.DeleteDraft(ctx, feedback.ReviewerID, feedback.RevieweeID, feedback.Period); delErr != nil {
			logger.Error("failed to delete draft after feedback submit", zap.Error(delErr))
		}
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
		logger.Error("failed to get team feedbacks", zap.Error(err))
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
		logger.Error("failed to get team members", zap.Error(err))
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	feedbacks, err := s.GetTeamFeedbacks(ctx, teamID)
	if err != nil {
		logger.Error("failed to get team feedbacks", zap.Error(err))
		return nil, fmt.Errorf("failed to get team feedbacks: %w", err)
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

// ExportFeedbacksForUser returns all feedbacks received by the user in the past 12 months,
// with reviewer names resolved for named (non-anonymous) entries.
func (s *FeedbackServiceImpl) ExportFeedbacksForUser(ctx context.Context, userID string) ([]*model.FeedbackExportEntry, error) {
	feedbacks, err := s.repo.GetFeedbacksByRevieweeID(ctx, userID, 1000, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get feedbacks for export: %w", err)
	}
	cutoff := time.Now().AddDate(-1, 0, 0)

	// Collect unique reviewer IDs for named feedbacks to batch-resolve names.
	reviewerIDs := make(map[string]struct{})
	var filtered []*model.Feedback
	for _, f := range feedbacks {
		if f.CreatedAt.After(cutoff) {
			filtered = append(filtered, f)
			if f.Visibility == "named" {
				reviewerIDs[f.ReviewerID] = struct{}{}
			}
		}
	}

	nameByID := make(map[string]string, len(reviewerIDs))
	for id := range reviewerIDs {
		if u, err := s.userRepo.GetUserByID(ctx, id); err == nil {
			nameByID[id] = u.Name
		}
	}

	result := make([]*model.FeedbackExportEntry, 0, len(filtered))
	for _, f := range filtered {
		entry := &model.FeedbackExportEntry{Feedback: f}
		if f.Visibility == "named" {
			entry.ReviewerName = nameByID[f.ReviewerID]
		}
		result = append(result, entry)
	}
	return result, nil
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
