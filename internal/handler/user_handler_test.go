package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
)

// ---------------------------------------------------------------------------
// GetTeammates handler tests
// ---------------------------------------------------------------------------

func TestUserHandler_GetTeammates_Success(t *testing.T) {
	now := time.Now()

	mockSvc := &mockUserService{
		GetTeammatesFn: func(_ context.Context, userID string) ([]*model.User, error) {
			assert.Equal(t, "user-123", userID)
			return []*model.User{
				{ID: "user-456", Name: "Bob", Email: "bob@example.com", Title: "Developer", CreatedAt: now, UpdatedAt: now},
				{ID: "user-789", Name: "Charlie", Email: "charlie@example.com", Title: "Designer", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	h := NewUserHandler(mockSvc)

	e := echo.New()
	e.GET("/me/teammates", func(c *echo.Context) error {
		c.Set("user_id", "user-123")
		return h.GetTeammates(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/teammates", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []userResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, 2)
	assert.Equal(t, "user-456", resp[0].ID)
	assert.Equal(t, "Bob", resp[0].Name)
	assert.Equal(t, "user-789", resp[1].ID)
	assert.Equal(t, "Charlie", resp[1].Name)
}

func TestUserHandler_GetTeammates_ServiceError(t *testing.T) {
	mockSvc := &mockUserService{
		GetTeammatesFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return nil, fmt.Errorf("user not found")
		},
	}

	h := NewUserHandler(mockSvc)

	e := echo.New()
	e.GET("/me/teammates", func(c *echo.Context) error {
		c.Set("user_id", "nonexistent")
		return h.GetTeammates(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/teammates", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "user not found")
}

func TestUserHandler_GetTeammates_EmptyTeam(t *testing.T) {
	mockSvc := &mockUserService{
		GetTeammatesFn: func(_ context.Context, _ string) ([]*model.User, error) {
			return []*model.User{}, nil
		},
	}

	h := NewUserHandler(mockSvc)

	e := echo.New()
	e.GET("/me/teammates", func(c *echo.Context) error {
		c.Set("user_id", "user-123")
		return h.GetTeammates(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/teammates", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []userResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp)
}
