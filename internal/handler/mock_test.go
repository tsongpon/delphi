package handler

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

// mockUserService implements UserService for handler tests.
type mockUserService struct {
	RegisterUserFn    func(ctx context.Context, user *model.User) (string, error)
	RegisterManagerFn func(ctx context.Context, user *model.User, teamName string) (string, error)
	RegisterMemberFn  func(ctx context.Context, user *model.User, teamID, role string) (string, error)
	LoginUserFn       func(ctx context.Context, email, password string) (string, error)
	GetTeammatesFn    func(ctx context.Context, userID string) ([]*model.User, error)
	UpdateUserRoleFn  func(ctx context.Context, userID, role string) error
}

func (m *mockUserService) RegisterUser(ctx context.Context, user *model.User) (string, error) {
	if m.RegisterUserFn != nil {
		return m.RegisterUserFn(ctx, user)
	}
	return "", nil
}

func (m *mockUserService) RegisterManager(ctx context.Context, user *model.User, teamName string) (string, error) {
	if m.RegisterManagerFn != nil {
		return m.RegisterManagerFn(ctx, user, teamName)
	}
	return "", nil
}

func (m *mockUserService) RegisterMember(ctx context.Context, user *model.User, teamID, role string) (string, error) {
	if m.RegisterMemberFn != nil {
		return m.RegisterMemberFn(ctx, user, teamID, role)
	}
	return "", nil
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

// mockInviteLinkService implements InviteLinkService for handler tests.
type mockInviteLinkService struct {
	CreateInviteLinkFn   func(ctx context.Context, teamID, createdBy string, expiresInDays int) (*model.InviteLink, string, error)
	ListLinksFn          func(ctx context.Context, teamID string) ([]*model.InviteLink, error)
	DeleteLinkFn         func(ctx context.Context, teamID, linkID string) error
	ValidateTokenFn      func(ctx context.Context, rawToken string) (*model.InviteLink, error)
	IncrementUsedCountFn func(ctx context.Context, id string) error
}

func (m *mockInviteLinkService) CreateInviteLink(ctx context.Context, teamID, createdBy string, expiresInDays int) (*model.InviteLink, string, error) {
	if m.CreateInviteLinkFn != nil {
		return m.CreateInviteLinkFn(ctx, teamID, createdBy, expiresInDays)
	}
	return nil, "", nil
}

func (m *mockInviteLinkService) ListLinks(ctx context.Context, teamID string) ([]*model.InviteLink, error) {
	if m.ListLinksFn != nil {
		return m.ListLinksFn(ctx, teamID)
	}
	return nil, nil
}

func (m *mockInviteLinkService) DeleteLink(ctx context.Context, teamID, linkID string) error {
	if m.DeleteLinkFn != nil {
		return m.DeleteLinkFn(ctx, teamID, linkID)
	}
	return nil
}

func (m *mockInviteLinkService) ValidateToken(ctx context.Context, rawToken string) (*model.InviteLink, error) {
	if m.ValidateTokenFn != nil {
		return m.ValidateTokenFn(ctx, rawToken)
	}
	return nil, nil
}

func (m *mockInviteLinkService) IncrementUsedCount(ctx context.Context, id string) error {
	if m.IncrementUsedCountFn != nil {
		return m.IncrementUsedCountFn(ctx, id)
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
