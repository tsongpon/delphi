package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/model"
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
func (s *UserServiceImpl) generateToken(user *model.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":     user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
		"team_id": user.TeamID,
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// RegisterUser creates a member with no team assignment (legacy / no-invite path) and returns a JWT.
func (s *UserServiceImpl) RegisterUser(ctx context.Context, user *model.User) (string, error) {
	user.ID = uuid.New().String()
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
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(createdUser)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// RegisterManager creates a new team and a manager user, then returns a JWT.
// Returns ErrTeamNameTaken if a team with that name already exists.
func (s *UserServiceImpl) RegisterManager(ctx context.Context, user *model.User, teamName string) (string, error) {
	existing, err := s.teamRepo.GetTeamByName(ctx, teamName)
	if err != nil {
		return "", fmt.Errorf("failed to check team name: %w", err)
	}
	if existing != nil {
		return "", ErrTeamNameTaken
	}

	now := time.Now()
	userID := uuid.New().String()

	team := &model.Team{
		ID:        uuid.New().String(),
		Name:      teamName,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	createdTeam, err := s.teamRepo.CreateTeam(ctx, team)
	if err != nil {
		return "", fmt.Errorf("failed to create team: %w", err)
	}

	user.ID = userID
	user.Role = "manager"
	user.TeamID = createdTeam.ID
	user.CreatedAt = now
	user.UpdatedAt = now

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(createdUser)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

// RegisterMember creates or updates a user for the given team via invite.
// If the email already exists with no team, the user is assigned to the team.
// Returns ErrEmailBelongsToDifferentTeam if the email is already on a different team.
func (s *UserServiceImpl) RegisterMember(ctx context.Context, user *model.User, teamID, role string) (string, error) {
	existing, err := s.repo.GetUserByEmail(ctx, user.Email)
	if err == nil && existing != nil {
		// Email already exists
		if existing.TeamID != "" && existing.TeamID != teamID {
			return "", ErrEmailBelongsToDifferentTeam
		}
		if err := s.repo.UpdateTeamID(ctx, existing.ID, teamID); err != nil {
			return "", fmt.Errorf("failed to assign team to existing user: %w", err)
		}
		existing.TeamID = teamID
		token, err := s.generateToken(existing)
		if err != nil {
			return "", fmt.Errorf("failed to sign token: %w", err)
		}
		return token, nil
	}

	// New user
	now := time.Now()
	user.ID = uuid.New().String()
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
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(createdUser)
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

	token, err := s.generateToken(user)
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
