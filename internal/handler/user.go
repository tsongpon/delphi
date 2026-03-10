package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

type UserHandler struct {
	UserService UserService
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		UserService: userService,
	}
}

func (h *UserHandler) GetTeammates(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	teammates, err := h.UserService.GetTeammates(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	response := make([]userResponse, 0, len(teammates))
	for _, t := range teammates {
		response = append(response, toUserResponse(t))
	}

	return c.JSON(http.StatusOK, response)
}
