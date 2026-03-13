package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository implements UserRepository for testing.
type mockUserRepository struct {
	CreateUserFn       func(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByEmailFn   func(ctx context.Context, email string) (*model.User, error)
	GetUserByIDFn      func(ctx context.Context, id string) (*model.User, error)
	GetUsersByTeamIDFn func(ctx context.Context, teamID string) ([]*model.User, error)
	UpdatePasswordFn   func(ctx context.Context, userID, hashedPassword string) error
	UpdateRoleFn       func(ctx context.Context, userID, role string) error
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	return m.CreateUserFn(ctx, user)
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return m.GetUserByEmailFn(ctx, email)
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return m.GetUserByIDFn(ctx, id)
}

func (m *mockUserRepository) GetUsersByTeamID(ctx context.Context, teamID string) ([]*model.User, error) {
	return m.GetUsersByTeamIDFn(ctx, teamID)
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	if m.UpdatePasswordFn != nil {
		return m.UpdatePasswordFn(ctx, userID, hashedPassword)
	}
	return nil
}

func (m *mockUserRepository) UpdateRole(ctx context.Context, userID, role string) error {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, userID, role)
	}
	return nil
}

// ---------------------------------------------------------------------------
// RegisterUser tests
// ---------------------------------------------------------------------------

func TestRegisterUser_Success(t *testing.T) {
	var captured *model.User

	repo := &mockUserRepository{
		CreateUserFn: func(_ context.Context, user *model.User) (*model.User, error) {
			captured = user
			return user, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	user := &model.User{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "plaintext123",
		Title:    "Engineer",
	}

	result, err := svc.RegisterUser(context.Background(), user)
	require.NoError(t, err)
	require.NotNil(t, result)

	// UUID was generated
	assert.NotEmpty(t, result.ID)

	// Timestamps were set
	assert.False(t, result.CreatedAt.IsZero())
	assert.False(t, result.UpdatedAt.IsZero())

	// Password was hashed (not plaintext)
	assert.NotEqual(t, "plaintext123", captured.Password)

	// Repo received the same user object
	assert.Equal(t, result.ID, captured.ID)
}

func TestRegisterUser_PasswordIsHashed(t *testing.T) {
	var capturedPassword string

	repo := &mockUserRepository{
		CreateUserFn: func(_ context.Context, user *model.User) (*model.User, error) {
			capturedPassword = user.Password
			return user, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	user := &model.User{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "my-secret-password",
		Title:    "Developer",
	}

	_, err := svc.RegisterUser(context.Background(), user)
	require.NoError(t, err)

	// Password is a bcrypt hash
	assert.True(t, strings.HasPrefix(capturedPassword, "$2a$"))

	// Hash matches the original plaintext
	err = bcrypt.CompareHashAndPassword([]byte(capturedPassword), []byte("my-secret-password"))
	assert.NoError(t, err)
}

func TestRegisterUser_RepoError(t *testing.T) {
	repo := &mockUserRepository{
		CreateUserFn: func(_ context.Context, _ *model.User) (*model.User, error) {
			return nil, fmt.Errorf("firestore unavailable")
		},
	}

	svc := NewUserService(repo, "test-secret")

	user := &model.User{
		Name:     "Charlie",
		Email:    "charlie@example.com",
		Password: "password",
		Title:    "QA",
	}

	result, err := svc.RegisterUser(context.Background(), user)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to create user")
}

// ---------------------------------------------------------------------------
// LoginUser tests
// ---------------------------------------------------------------------------

func TestLoginUser_Success(t *testing.T) {
	hashedPw, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &mockUserRepository{
		GetUserByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{
				ID:       "user-123",
				Name:     "Alice",
				Email:    email,
				Password: string(hashedPw),
				Title:    "Engineer",
			}, nil
		},
	}

	secret := "test-jwt-secret"
	svc := NewUserService(repo, secret)

	token, err := svc.LoginUser(context.Background(), "alice@example.com", "correct-password")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse and validate token claims
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)

	assert.Equal(t, "user-123", claims["sub"])
	assert.Equal(t, "alice@example.com", claims["email"])
	assert.Equal(t, "Alice", claims["name"])

	// exp should be approximately 24 hours from now
	exp, ok := claims["exp"].(float64)
	require.True(t, ok)
	expectedExp := float64(time.Now().Add(24 * time.Hour).Unix())
	assert.InDelta(t, expectedExp, exp, 5)

	// iat should be approximately now
	iat, ok := claims["iat"].(float64)
	require.True(t, ok)
	assert.InDelta(t, float64(time.Now().Unix()), iat, 5)
}

func TestLoginUser_UserNotFound(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	svc := NewUserService(repo, "test-secret")

	token, err := svc.LoginUser(context.Background(), "nobody@example.com", "password")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.EqualError(t, err, "invalid credentials")
}

func TestLoginUser_WrongPassword(t *testing.T) {
	hashedPw, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	require.NoError(t, err)

	repo := &mockUserRepository{
		GetUserByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return &model.User{
				ID:       "user-123",
				Name:     "Alice",
				Email:    "alice@example.com",
				Password: string(hashedPw),
			}, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	token, err := svc.LoginUser(context.Background(), "alice@example.com", "wrong-password")
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.EqualError(t, err, "invalid credentials")
}

// ---------------------------------------------------------------------------
// GetTeammates tests
// ---------------------------------------------------------------------------

func TestGetTeammates_Success(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Alice", TeamID: "team-1"}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return []*model.User{
				{ID: "user-1", Name: "Alice", TeamID: "team-1"},
				{ID: "user-2", Name: "Bob", TeamID: "team-1"},
				{ID: "user-3", Name: "Charlie", TeamID: "team-1"},
			}, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	result, err := svc.GetTeammates(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Requesting user should be filtered out
	for _, u := range result {
		assert.NotEqual(t, "user-1", u.ID)
	}
}

func TestGetTeammates_NoTeam(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Alice", TeamID: ""}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			t.Fatal("GetUsersByTeamID should not be called when user has no team")
			return nil, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	result, err := svc.GetTeammates(context.Background(), "user-1")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestGetTeammates_UserNotFound(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, fmt.Errorf("user not found")
		},
	}

	svc := NewUserService(repo, "test-secret")

	result, err := svc.GetTeammates(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get user")
}

func TestGetTeammates_RepoError(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	svc := NewUserService(repo, "test-secret")

	result, err := svc.GetTeammates(context.Background(), "user-1")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "failed to get teammates")
}

func TestGetTeammates_OnlyUserOnTeam(t *testing.T) {
	repo := &mockUserRepository{
		GetUserByIDFn: func(_ context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, TeamID: "team-1"}, nil
		},
		GetUsersByTeamIDFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return []*model.User{
				{ID: "user-1", Name: "Alice", TeamID: "team-1"},
			}, nil
		},
	}

	svc := NewUserService(repo, "test-secret")

	result, err := svc.GetTeammates(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}
