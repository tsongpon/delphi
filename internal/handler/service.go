package handler

import (
	"context"
	"time"

	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type UserService interface {
	RegisterUser(ctx context.Context, user *model.User) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (string, error)
	GetTeammates(ctx context.Context, userID string) ([]*model.User, error)
	UpdateUserRole(ctx context.Context, userID, role string) error
}

type FeedbackService interface {
	CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error)
	GetGivenFeedbacksForUser(ctx context.Context, userID string, limit int, cursor string) ([]*model.Feedback, error)
	GetTeamFeedbacks(ctx context.Context, teamID string) ([]*model.Feedback, error)
	GetTeamDashboard(ctx context.Context, teamID string) (*service.TeamDashboard, error)
	GetFeedbacksForMember(ctx context.Context, teamID, memberID string, limit int, cursor string) ([]*model.Feedback, error)
}

type TeamService interface {
	CreateTeam(ctx context.Context, name string) (*model.Team, error)
}

type PasswordResetService interface {
	GenerateResetLink(ctx context.Context, userID string) (resetLink string, expiresAt time.Time, err error)
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}
