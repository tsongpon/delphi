package handler

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
)

type UserService interface {
	RegisterUser(ctx context.Context, user *model.User) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (string, error)
	GetTeammates(ctx context.Context, userID string) ([]*model.User, error)
}

type FeedbackService interface {
	CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
}
