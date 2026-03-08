package service

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUsersByTeamID(ctx context.Context, teamID string) ([]*model.User, error)
}

type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedback(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error)
}
