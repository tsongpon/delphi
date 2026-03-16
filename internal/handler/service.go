package handler

import (
	"context"
	"time"

	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type UserService interface {
	RegisterUser(ctx context.Context, user *model.User) (string, error)
	RegisterManager(ctx context.Context, user *model.User, teamName string) (string, error)
	RegisterMember(ctx context.Context, user *model.User, teamID, role string) (string, error)
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
	ExportFeedbacksForUser(ctx context.Context, userID string) ([]*service.FeedbackExportEntry, error)
}

type TeamService interface {
	CreateTeam(ctx context.Context, name string) (*model.Team, error)
}

type PasswordResetService interface {
	GenerateResetLink(ctx context.Context, userID string) (resetLink string, expiresAt time.Time, err error)
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}

type InviteLinkService interface {
	CreateInviteLink(ctx context.Context, teamID, createdBy string, expiresInDays int) (*model.InviteLink, string, error)
	ListLinks(ctx context.Context, teamID string) ([]*model.InviteLink, error)
	DeleteLink(ctx context.Context, teamID, linkID string) error
	ValidateToken(ctx context.Context, rawToken string) (*model.InviteLink, error)
	IncrementUsedCount(ctx context.Context, id string) error
}
