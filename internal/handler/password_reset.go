package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/service"
)

type PasswordResetHandler struct {
	PasswordResetService PasswordResetService
}

func NewPasswordResetHandler(svc PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{PasswordResetService: svc}
}

// GenerateResetLink is an admin-only endpoint that creates a one-time reset link for the given user.
// POST /admin/users/:userID/reset-link
func (h *PasswordResetHandler) GenerateResetLink(c *echo.Context) error {
	userID := c.Param("userID")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "userID is required"})
	}

	ctx := c.Request().Context()
	resetLink, expiresAt, err := h.PasswordResetService.GenerateResetLink(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate reset link"})
	}

	return c.JSON(http.StatusOK, generateResetLinkResponse{
		ResetLink: resetLink,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// ResetPassword is a public endpoint that accepts a token and sets a new password.
// POST /reset-password
func (h *PasswordResetHandler) ResetPassword(c *echo.Context) error {
	var req resetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Token == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token and new_password are required"})
	}

	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "new_password must be at least 8 characters"})
	}

	ctx := c.Request().Context()
	if err := h.PasswordResetService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidResetToken) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid or expired reset token"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to reset password"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "password reset successfully"})
}
