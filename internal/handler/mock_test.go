package handler

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
)

// mockUserService implements UserService for handler tests.
type mockUserService struct {
	RegisterUserFn func(ctx context.Context, user *model.User) (*model.User, error)
	LoginUserFn    func(ctx context.Context, email, password string) (string, error)
	GetTeammatesFn func(ctx context.Context, userID string) ([]*model.User, error)
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

// mockFeedbackService implements FeedbackService for handler tests.
type mockFeedbackService struct {
	CreateFeedbackFn func(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
}

func (m *mockFeedbackService) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	return m.CreateFeedbackFn(ctx, feedback)
}
