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

// UserServiceImpl implements handler.UserService.
type UserServiceImpl struct {
	repo      UserRepository
	jwtSecret []byte
}

// NewUserService creates a new UserServiceImpl.
func NewUserService(repo UserRepository, jwtSecret string) *UserServiceImpl {
	return &UserServiceImpl{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// RegisterUser generates an ID, hashes the password, and persists the user.
func (s *UserServiceImpl) RegisterUser(ctx context.Context, user *model.User) (*model.User, error) {
	user.ID = uuid.New().String()

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return createdUser, nil
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

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"iat":   now.Unix(),
		"exp":   now.Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
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
