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
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
	UpdateRole(ctx context.Context, userID, role string) error
}

type TokenRepository interface {
	SaveToken(ctx context.Context, token *model.PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error)
	DeleteToken(ctx context.Context, tokenHash string) error
}

type TeamRepository interface {
	CreateTeam(ctx context.Context, team *model.Team) (*model.Team, error)
}

type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedback(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error)
	GetFeedbacksByRevieweeID(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerID(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDs(ctx context.Context, reviewerIDs []string) ([]*model.Feedback, error)
}
