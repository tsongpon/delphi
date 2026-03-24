package service

import (
	"context"
	"time"

	"github.com/tsongpon/delphi/internal/model"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	GetUsersByTeamID(ctx context.Context, teamID string) ([]*model.User, error)
	GetAllUsers(ctx context.Context) ([]*model.User, error)
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
	UpdateRole(ctx context.Context, userID, role string) error
	UpdateTeamID(ctx context.Context, userID, teamID string) error
	DeleteUser(ctx context.Context, userID string) error
}

type TokenRepository interface {
	SaveToken(ctx context.Context, token *model.PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error)
	DeleteToken(ctx context.Context, tokenHash string) error
}

type TeamRepository interface {
	CreateTeam(ctx context.Context, team *model.Team) (*model.Team, error)
	GetTeamByID(ctx context.Context, teamID string) (*model.Team, error)
	GetTeamByName(ctx context.Context, name string) (*model.Team, error)
}

type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error)
	GetFeedback(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error)
	GetFeedbacksByRevieweeID(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerID(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error)
	GetFeedbacksByReviewerIDs(ctx context.Context, reviewerIDs []string) ([]*model.Feedback, error)
	GetFeedbacksByRevieweeIDSince(ctx context.Context, revieweeID string, since time.Time) ([]*model.Feedback, error)
}

type EmailSender interface {
	SendFeedbackDigest(ctx context.Context, toName, toEmail string, count int) error
}

type InviteLinkRepository interface {
	CreateInviteLink(ctx context.Context, link *model.InviteLink) (*model.InviteLink, error)
	GetByID(ctx context.Context, id string) (*model.InviteLink, error)
	GetByTeamID(ctx context.Context, teamID string) ([]*model.InviteLink, error)
	DeleteInviteLink(ctx context.Context, id string) error
	IncrementUsedCount(ctx context.Context, id string) error
}

type FeedbackPeriodRepository interface {
	CreatePeriod(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error)
	GetActivePeriodForTeam(ctx context.Context, teamID string, now time.Time) (*model.FeedbackPeriod, error)
	ListPeriodsForTeam(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error)
	DeletePeriod(ctx context.Context, teamID, periodID string) error
}
