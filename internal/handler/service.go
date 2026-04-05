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
	ExportFeedbacksForUser(ctx context.Context, userID string) ([]*model.FeedbackExportEntry, error)
}

type TeamService interface {
	CreateTeam(ctx context.Context, name string) (*model.Team, error)
}

type PasswordResetService interface {
	GenerateResetLink(ctx context.Context, userID string) (resetLink string, expiresAt time.Time, err error)
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
	ForgotPassword(ctx context.Context, email string) error
}

type NotifyService interface {
	// SendFeedbackDigest sends digest emails for feedback received yesterday.
	// Pass a non-empty teamID to restrict notifications to that team only.
	SendFeedbackDigest(ctx context.Context, teamID string) (*service.NotifyResult, error)
}

type InviteLinkService interface {
	CreateInviteLink(ctx context.Context, teamID, createdBy string, expiresInDays int) (*model.InviteLink, string, error)
	ListLinks(ctx context.Context, teamID string) ([]*model.InviteLink, error)
	DeleteLink(ctx context.Context, teamID, linkID string) error
	ValidateToken(ctx context.Context, rawToken string) (*model.InviteLink, error)
	IncrementUsedCount(ctx context.Context, id string) error
}

type FeedbackDraftService interface {
	SaveDraft(ctx context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error)
	GetDraft(ctx context.Context, reviewerID, revieweeID string) (*model.FeedbackDraft, error)
	GetMyDrafts(ctx context.Context, reviewerID string) ([]*model.FeedbackDraft, error)
	DeleteDraft(ctx context.Context, reviewerID, revieweeID string) error
}

type FeedbackPeriodService interface {
	CreatePeriod(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error)
	GetActivePeriodForTeam(ctx context.Context, teamID string) (*model.FeedbackPeriod, error)
	ListPeriodsForTeam(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error)
	DeletePeriod(ctx context.Context, teamID, periodID string) error
}
