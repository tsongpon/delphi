package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/apperr"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var errInvalidCredentials = fmt.Errorf("invalid credentials")
var ErrTeamNameTaken = fmt.Errorf("team name already taken")
var ErrEmailBelongsToDifferentTeam = fmt.Errorf("email already belongs to a different team")

// UserServiceImpl implements handler.UserService.
type UserServiceImpl struct {
	repo      UserRepository
	teamRepo  TeamRepository
	jwtSecret []byte
}

// NewUserService creates a new UserServiceImpl.
func NewUserService(repo UserRepository, teamRepo TeamRepository, jwtSecret string) *UserServiceImpl {
	return &UserServiceImpl{
		repo:      repo,
		teamRepo:  teamRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// generateToken creates a signed JWT for the given user.
func (s *UserServiceImpl) generateToken(ctx context.Context, user *model.User) (string, error) {
	teamName := ""
	if user.TeamID != "" {
		if team, err := s.teamRepo.GetTeamByID(ctx, user.TeamID); err == nil {
			teamName = team.Name
		}
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       user.ID,
		"email":     user.Email,
		"name":      user.Name,
		"title":     user.Title,
		"role":      user.Role,
		"team_id":   user.TeamID,
		"team_name": teamName,
		"iat":       now.Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// RegisterUser creates a member with no team assignment (legacy / no-invite path) and returns a JWT.
func (s *UserServiceImpl) RegisterUser(ctx context.Context, user *model.User) (string, error) {
	user.ID = hashEmail(user.Email)
	user.Role = "member"

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		if e, ok := err.(*apperr.DuplicateResourceError); ok {
			logger.Error("failed to create user, duplicate resource", zap.Error(e))
			return "", err
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(ctx, createdUser)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// RegisterManager creates a new team and a manager user, then returns a JWT.
// Returns ErrTeamNameTaken if a team with that name already exists.
func (s *UserServiceImpl) RegisterManager(ctx context.Context, user *model.User, teamName string) (string, error) {
	now := time.Now()
	userID := hashEmail(user.Email)

	teamID := uuid.New().String()

	user.ID = userID
	user.Role = "manager"
	user.TeamID = teamID
	user.CreatedAt = now
	user.UpdatedAt = now

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		if e, ok := err.(*apperr.DuplicateResourceError); ok {
			logger.Error("failed to create user, duplicate resource", zap.Error(e))
			return "", err
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	team := &model.Team{
		ID:        teamID,
		Name:      teamName,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.teamRepo.CreateTeam(ctx, team)
	if err != nil {
		logger.Error("failed to create team", zap.Error(err))
		err := s.repo.DeleteUser(ctx, user.ID)
		if err != nil {
			logger.Error("failed to delete user", zap.Error(err))
		}
		return "", fmt.Errorf("failed to create team: %w", err)
	}

	token, err := s.generateToken(ctx, createdUser)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// RegisterMember creates or updates a user for the given team via invite.
// If the email already exists with no team, the user is assigned to the team.
// Returns ErrEmailBelongsToDifferentTeam if the email is already on a different team.
func (s *UserServiceImpl) RegisterMember(ctx context.Context, user *model.User, teamID, role string) (string, error) {
	// New user
	now := time.Now()
	user.ID = hashEmail(user.Email)
	user.Role = role
	user.TeamID = teamID
	user.CreatedAt = now
	user.UpdatedAt = now

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		if e, ok := err.(*apperr.DuplicateResourceError); ok {
			logger.Error("failed to create user, duplicate resource", zap.Error(e))
			return "", err
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(ctx, createdUser)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// LoginUser validates credentials and returns a signed JWT token.
func (s *UserServiceImpl) LoginUser(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errInvalidCredentials
	}

	token, err := s.generateToken(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// UpdateUserRole updates the role of a user.
func (s *UserServiceImpl) UpdateUserRole(ctx context.Context, userID, role string) error {
	return s.repo.UpdateRole(ctx, userID, role)
}

// GetTeammates returns all users sharing the same team as the given user, excluding the user themselves.
func (s *UserServiceImpl) GetTeammates(ctx context.Context, userID string) ([]*model.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.TeamID == "" {
		return []*model.User{}, nil
	}

	teammates, err := s.repo.GetUsersByTeamID(ctx, user.TeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teammates: %w", err)
	}

	// Filter out the requesting user
	result := make([]*model.User, 0, len(teammates))
	for _, t := range teammates {
		if t.ID != userID {
			result = append(result, t)
		}
	}

	return result, nil
}

func hashEmail(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("%x", h)
}
