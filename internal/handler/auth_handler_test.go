package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
)

// ---------------------------------------------------------------------------
// RegisterUser handler tests
// ---------------------------------------------------------------------------

func TestAuthHandler_RegisterUser_Success(t *testing.T) {
	now := time.Now()

	mockSvc := &mockUserService{
		RegisterUserFn: func(_ context.Context, user *model.User) (*model.User, error) {
			user.ID = "generated-uuid"
			user.CreatedAt = now
			user.UpdatedAt = now
			return user, nil
		},
	}

	h := NewAuthHandler(mockSvc)

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123","title":"Engineer"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.RegisterUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp userResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "generated-uuid", resp.ID)
	assert.Equal(t, "Alice", resp.Name)
	assert.Equal(t, "alice@example.com", resp.Email)
	assert.Equal(t, "Engineer", resp.Title)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)

	// Password must NOT appear in the response
	assert.NotContains(t, rec.Body.String(), "secret123")
}

func TestAuthHandler_RegisterUser_InvalidBody(t *testing.T) {
	mockSvc := &mockUserService{}
	h := NewAuthHandler(mockSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.RegisterUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestAuthHandler_RegisterUser_ServiceError(t *testing.T) {
	mockSvc := &mockUserService{
		RegisterUserFn: func(_ context.Context, _ *model.User) (*model.User, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	h := NewAuthHandler(mockSvc)

	body := `{"name":"Alice","email":"alice@example.com","password":"secret123","title":"Engineer"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.RegisterUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to register user")
}

// ---------------------------------------------------------------------------
// LoginUser handler tests
// ---------------------------------------------------------------------------

func TestAuthHandler_LoginUser_Success(t *testing.T) {
	mockSvc := &mockUserService{
		LoginUserFn: func(_ context.Context, _ string, _ string) (string, error) {
			return "jwt-token-string", nil
		},
	}

	h := NewAuthHandler(mockSvc)

	body := `{"email":"alice@example.com","password":"secret123"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.LoginUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp loginResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "jwt-token-string", resp.Token)
}

func TestAuthHandler_LoginUser_InvalidBody(t *testing.T) {
	mockSvc := &mockUserService{}
	h := NewAuthHandler(mockSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.LoginUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestAuthHandler_LoginUser_InvalidCredentials(t *testing.T) {
	mockSvc := &mockUserService{
		LoginUserFn: func(_ context.Context, _ string, _ string) (string, error) {
			return "", fmt.Errorf("invalid credentials")
		},
	}

	h := NewAuthHandler(mockSvc)

	body := `{"email":"alice@example.com","password":"wrong"}`
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.LoginUser(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid credentials")
}
