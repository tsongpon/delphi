package handler

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type AdminHandler struct {
	UserService UserService
	TeamService TeamService
}

func NewAdminHandler(userService UserService, teamService TeamService) *AdminHandler {
	return &AdminHandler{UserService: userService, TeamService: teamService}
}

func (h *AdminHandler) UpdateUserRole(c *echo.Context) error {
	userID := c.Param("userID")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing userID"})
	}

	var req updateRoleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Role != "member" && req.Role != "manager" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be 'member' or 'manager'"})
	}

	ctx := c.Request().Context()
	if err := h.UserService.UpdateUserRole(ctx, userID, req.Role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update role"})
	}

	return c.JSON(http.StatusOK, map[string]string{"role": req.Role})
}

func (h *AdminHandler) CreateTeam(c *echo.Context) error {
	var req createTeamRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	ctx := c.Request().Context()
	team, err := h.TeamService.CreateTeam(ctx, req.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create team"})
	}

	return c.JSON(http.StatusCreated, toTeamResponse(team))
}
