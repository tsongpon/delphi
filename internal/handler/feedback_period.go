package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type FeedbackPeriodHandler struct {
	PeriodService FeedbackPeriodService
}

func NewFeedbackPeriodHandler(periodService FeedbackPeriodService) *FeedbackPeriodHandler {
	return &FeedbackPeriodHandler{PeriodService: periodService}
}

// CreatePeriod handles POST /teams/:teamId/periods
func (h *FeedbackPeriodHandler) CreatePeriod(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}
	callerID, _ := c.Get("user_id").(string)

	var req createFeedbackPeriodRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Name == "" || req.StartDate == "" || req.EndDate == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name, start_date, and end_date are required"})
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid start_date format, use RFC3339"})
	}
	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid end_date format, use RFC3339"})
	}

	period := &model.FeedbackPeriod{
		TeamID:    teamID,
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedBy: callerID,
	}

	ctx := c.Request().Context()
	created, err := h.PeriodService.CreatePeriod(ctx, period)
	if err != nil {
		if errors.Is(err, service.ErrPeriodEndBeforeStart) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrPeriodNameExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create period"})
	}

	return c.JSON(http.StatusCreated, toFeedbackPeriodResponse(created))
}

// ListPeriods handles GET /teams/:teamId/periods
func (h *FeedbackPeriodHandler) ListPeriods(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	ctx := c.Request().Context()
	periods, err := h.PeriodService.ListPeriodsForTeam(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list periods"})
	}

	resp := make([]feedbackPeriodResponse, 0, len(periods))
	for _, p := range periods {
		resp = append(resp, toFeedbackPeriodResponse(p))
	}
	return c.JSON(http.StatusOK, resp)
}

// DeletePeriod handles DELETE /teams/:teamId/periods/:periodId
func (h *FeedbackPeriodHandler) DeletePeriod(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	periodID := c.Param("periodId")
	ctx := c.Request().Context()
	if err := h.PeriodService.DeletePeriod(ctx, teamID, periodID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete period"})
	}
	return c.NoContent(http.StatusNoContent)
}

// GetActivePeriod handles GET /teams/:teamId/periods/active
func (h *FeedbackPeriodHandler) GetActivePeriod(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	ctx := c.Request().Context()
	period, err := h.PeriodService.GetActivePeriodForTeam(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get active period"})
	}
	if period == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "no active feedback period"})
	}
	return c.JSON(http.StatusOK, toFeedbackPeriodResponse(period))
}
