package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
)

type AuthHandler struct {
	UserService UserService
}

func NewAuthHandler(userService UserService) *AuthHandler {
	return &AuthHandler{
		UserService: userService,
	}
}

func (h *AuthHandler) RegisterUser(c *echo.Context) error {
	var req registerUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	user := &model.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Title:    req.Title,
	}

	ctx := c.Request().Context()
	createdUser, err := h.UserService.RegisterUser(ctx, user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to register user"})
	}

	return c.JSON(http.StatusCreated, toUserResponse(createdUser))
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
