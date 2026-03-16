package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/apperr"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	UserService       UserService
	InviteLinkService InviteLinkService
}

func NewAuthHandler(userService UserService, inviteLinkService InviteLinkService) *AuthHandler {
	return &AuthHandler{
		UserService:       userService,
		InviteLinkService: inviteLinkService,
	}
}

func (h *AuthHandler) RegisterUser(c *echo.Context) error {
	logger.Debug("start register user")
	var req registerUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	ctx := c.Request().Context()

	switch {
	case req.InviteToken != "":
		// Member registration via invite link
		link, err := h.InviteLinkService.ValidateToken(ctx, req.InviteToken)
		if err != nil {
			if errors.Is(err, service.ErrInviteLinkExpired) {
				return c.JSON(http.StatusGone, map[string]string{"error": "invite token expired"})
			}
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite token"})
		}

		user := &model.User{
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Title:    req.Title,
		}
		token, err := h.UserService.RegisterMember(ctx, user, link.TeamID, link.Role)
		if err != nil {
			if errors.Is(err, service.ErrEmailBelongsToDifferentTeam) {
				return c.JSON(http.StatusConflict, map[string]string{"error": "email already belongs to a different team"})
			}
			if e, ok := err.(*apperr.DuplicateResourceError); ok {
				logger.Error("failed to register user, duplicate resource", zap.Error(e))
				return c.JSON(http.StatusConflict, map[string]string{"error": "email already belongs to a different user"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to register user"})
		}

		_ = h.InviteLinkService.IncrementUsedCount(ctx, link.ID)

		return c.JSON(http.StatusCreated, loginResponse{Token: token})

	case req.TeamName != "":
		// Manager registration — creates a new team
		user := &model.User{
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Title:    req.Title,
		}
		token, err := h.UserService.RegisterManager(ctx, user, req.TeamName)
		if err != nil {
			if errors.Is(err, service.ErrTeamNameTaken) {
				return c.JSON(http.StatusConflict, map[string]string{"error": "team name already taken"})
			}
			if e, ok := err.(*apperr.DuplicateResourceError); ok {
				logger.Error("failed to register user, duplicate resource", zap.Error(e))
				return c.JSON(http.StatusConflict, map[string]string{"error": "email already belongs to a different user"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to register user"})
		}
		return c.JSON(http.StatusCreated, loginResponse{Token: token})

	default:
		// Legacy registration — member with no team
		user := &model.User{
			Name:     req.Name,
			Email:    req.Email,
			Password: req.Password,
			Title:    req.Title,
		}
		token, err := h.UserService.RegisterUser(ctx, user)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to register user"})
		}
		return c.JSON(http.StatusCreated, loginResponse{Token: token})
	}
}

func (h *AuthHandler) LoginUser(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	ctx := c.Request().Context()
	token, err := h.UserService.LoginUser(ctx, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	return c.JSON(http.StatusOK, loginResponse{Token: token})
}
