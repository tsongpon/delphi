package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

// generateTestToken creates a signed JWT for testing.
func generateTestToken(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// validClaims returns standard valid claims for testing.
func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub":   "user-123",
		"email": "test@example.com",
		"name":  "Test User",
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	token := generateTestToken(t, validClaims(), testSecret)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "ok")
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.True(t, nextCalled)
	assert.Equal(t, "user-123", c.Get("user_id"))
	assert.Equal(t, "test@example.com", c.Get("email"))
	assert.Equal(t, "Test User", c.Get("name"))
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "missing authorization header", resp["error"])
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid authorization format")
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":   "user-123",
		"email": "test@example.com",
		"name":  "Test User",
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		"exp":   time.Now().Add(-1 * time.Hour).Unix(), // expired 1 hour ago
	}
	token := generateTestToken(t, claims, testSecret)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired token")
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	token := generateTestToken(t, validClaims(), "wrong-secret")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid or expired token")
}

func TestJWTAuth_MalformedToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err := handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_WrongSigningMethod(t *testing.T) {
	// Create a token with "none" signing method to test alg validation
	claims := validClaims()
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signed))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c *echo.Context) error {
		nextCalled = true
		return nil
	}

	mw := JWTAuth(testSecret)
	handler := mw(next)

	err = handler(c)
	require.NoError(t, err)

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
