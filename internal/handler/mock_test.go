package handler

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

// mockUserService implements UserService for handler tests.
type mockUserService struct {
	RegisterUserFn   func(ctx context.Context, user *model.User) (*model.User, error)
	LoginUserFn      func(ctx context.Context, email, password string) (string, error)
	GetTeammatesFn   func(ctx context.Context, userID string) ([]*model.User, error)
	UpdateUserRoleFn func(ctx context.Context, userID, role string) error
}

func (m *mockUserService) RegisterUser(ctx context.Context, user *model.User) (*model.User, error) {
	return m.RegisterUserFn(ctx, user)
}

func (m *mockUserService) LoginUser(ctx context.Context, email, password string) (string, error) {
	return m.LoginUserFn(ctx, email, password)
}

func (m *mockUserService) GetTeammates(ctx context.Context, userID string) ([]*model.User, error) {
	return m.GetTeammatesFn(ctx, userID)
}

func (m *mockUserService) UpdateUserRole(ctx context.Context, userID, role string) error {
	if m.UpdateUserRoleFn != nil {
		return m.UpdateUserRoleFn(ctx, userID, role)
	}
	return nil
}

// mockFeedbackService implements FeedbackService for handler tests.
type mockFeedbackService struct {
	CreateFeedbackFn           func(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedbacksForUserFn      func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error)
	GetGivenFeedbacksForUserFn func(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error)
	GetTeamFeedbacksFn         func(ctx context.Context, teamID string) ([]*model.Feedback, error)
	GetTeamDashboardFn         func(ctx context.Context, teamID string) (*service.TeamDashboard, error)
	GetFeedbacksForMemberFn    func(ctx context.Context, teamID, memberID string, limit int, cursor string) ([]*model.Feedback, error)
}

func (m *mockFeedbackService) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	return m.CreateFeedbackFn(ctx, feedback)
}

func (m *mockFeedbackService) GetFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error) {
	return m.GetFeedbacksForUserFn(ctx, userID, limit, cursor)
}

func (m *mockFeedbackService) GetGivenFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error) {
	return m.GetGivenFeedbacksForUserFn(ctx, userID, limit, cursor)
}

func (m *mockFeedbackService) GetTeamFeedbacks(ctx context.Context, teamID string) ([]*model.Feedback, error) {
	if m.GetTeamFeedbacksFn != nil {
		return m.GetTeamFeedbacksFn(ctx, teamID)
	}
	return nil, nil
}

func (m *mockFeedbackService) GetTeamDashboard(ctx context.Context, teamID string) (*service.TeamDashboard, error) {
	if m.GetTeamDashboardFn != nil {
		return m.GetTeamDashboardFn(ctx, teamID)
	}
	return &service.TeamDashboard{}, nil
}

func (m *mockFeedbackService) GetFeedbacksForMember(ctx context.Context, teamID, memberID string, limit int, cursor string) ([]*model.Feedback, error) {
	if m.GetFeedbacksForMemberFn != nil {
		return m.GetFeedbacksForMemberFn(ctx, teamID, memberID, limit, cursor)
	}
	return nil, nil
}
