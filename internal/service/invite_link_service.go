package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

var ErrInviteLinkExpired = fmt.Errorf("invite link expired")
var ErrInviteLinkInvalid = fmt.Errorf("invite link invalid")
var ErrInviteLinkNotFound = fmt.Errorf("invite link not found")

// InviteLinkServiceImpl implements handler.InviteLinkService.
type InviteLinkServiceImpl struct {
	repo       InviteLinkRepository
	teamRepo   TeamRepository
	jwtSecret  []byte
	appBaseURL string
}

// NewInviteLinkService creates a new InviteLinkServiceImpl.
func NewInviteLinkService(repo InviteLinkRepository, teamRepo TeamRepository, jwtSecret, appBaseURL string) *InviteLinkServiceImpl {
	return &InviteLinkServiceImpl{
		repo:       repo,
		teamRepo:   teamRepo,
		jwtSecret:  []byte(jwtSecret),
		appBaseURL: appBaseURL,
	}
}

// CreateInviteLink generates a signed invite JWT, stores the link in Firestore, and returns it with the full URL.
func (s *InviteLinkServiceImpl) CreateInviteLink(ctx context.Context, teamID, createdBy string, expiresInDays int) (*model.InviteLink, string, error) {
	if expiresInDays <= 0 {
		expiresInDays = 7
	}

	team, err := s.teamRepo.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get team: %w", err)
	}
	if team == nil {
		return nil, "", fmt.Errorf("team not found")
	}

	linkID := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresInDays) * 24 * time.Hour)

	claims := jwt.MapClaims{
		"jti":       linkID,
		"team_id":   teamID,
		"team_name": team.Name,
		"role":      "member",
		"exp":       expiresAt.Unix(),
		"iat":       now.Unix(),
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := rawToken.SignedString(s.jwtSecret)
	if err != nil {
		logger.Error("failed to sign invite token", zap.Error(err))
		return nil, "", fmt.Errorf("failed to sign invite token: %w", err)
	}

	link := &model.InviteLink{
		ID:        linkID,
		Token:     signedToken,
		TeamID:    teamID,
		TeamName:  team.Name,
		CreatedBy: createdBy,
		Role:      "member",
		ExpiresAt: expiresAt,
		UsedCount: 0,
		CreatedAt: now,
	}

	created, err := s.repo.CreateInviteLink(ctx, link)
	if err != nil {
		logger.Error("failed to create invite link", zap.Error(err))
		return nil, "", err
	}

	inviteURL := fmt.Sprintf("%s/register?token=%s", s.appBaseURL, signedToken)
	return created, inviteURL, nil
}

// ListLinks returns all invite links for a team.
func (s *InviteLinkServiceImpl) ListLinks(ctx context.Context, teamID string) ([]*model.InviteLink, error) {
	links, err := s.repo.GetByTeamID(ctx, teamID)
	if err != nil {
		logger.Error("failed to list invite links", zap.Error(err))
		return nil, err
	}
	return links, nil
}

// DeleteLink hard-deletes a specific invite link from Firestore.
// Returns ErrInviteLinkNotFound if the link doesn't exist or belongs to a different team.
func (s *InviteLinkServiceImpl) DeleteLink(ctx context.Context, teamID, linkID string) error {
	link, err := s.repo.GetByID(ctx, linkID)
	if err != nil {
		logger.Error("failed to get invite link", zap.Error(err))
		return err
	}
	if link == nil || link.TeamID != teamID {
		return ErrInviteLinkNotFound
	}
	err = s.repo.DeleteInviteLink(ctx, linkID)
	if err != nil {
		logger.Error("failed to delete invite link", zap.Error(err))
		return err
	}
	return nil
}

// ValidateToken parses and validates an invite JWT, then checks Firestore that the record still exists.
// Returns ErrInviteLinkInvalid for bad signatures or deleted links, ErrInviteLinkExpired for expired tokens.
func (s *InviteLinkServiceImpl) ValidateToken(ctx context.Context, rawToken string) (*model.InviteLink, error) {
	token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrInviteLinkExpired
		}
		return nil, ErrInviteLinkInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInviteLinkInvalid
	}

	linkID, _ := claims["jti"].(string)
	if linkID == "" {
		return nil, ErrInviteLinkInvalid
	}

	link, err := s.repo.GetByID(ctx, linkID)
	if err != nil {
		return nil, err
	}
	if link == nil {
		// Record was deleted — treat as invalid
		return nil, ErrInviteLinkInvalid
	}
	if time.Now().After(link.ExpiresAt) {
		return nil, ErrInviteLinkExpired
	}

	return link, nil
}

// IncrementUsedCount increments the used_count of an invite link.
func (s *InviteLinkServiceImpl) IncrementUsedCount(ctx context.Context, id string) error {
	err := s.repo.IncrementUsedCount(ctx, id)
	if err != nil {
		logger.Error("failed to increment used count", zap.Error(err))
		return err
	}
	return nil
}
