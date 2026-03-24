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
	"github.com/tsongpon/delphi/internal/service"
)

// ---------------------------------------------------------------------------
// Route helpers
// ---------------------------------------------------------------------------

func setupCreatePeriodRoute(e *echo.Echo, h *FeedbackPeriodHandler, teamID, callerTeamID, callerID string) {
	e.POST("/teams/:teamId/periods", func(c *echo.Context) error {
		c.Set("team_id", callerTeamID)
		c.Set("user_id", callerID)
		return h.CreatePeriod(c)
	})
}

func setupListPeriodsRoute(e *echo.Echo, h *FeedbackPeriodHandler, callerTeamID string) {
	e.GET("/teams/:teamId/periods", func(c *echo.Context) error {
		c.Set("team_id", callerTeamID)
		return h.ListPeriods(c)
	})
}

func setupDeletePeriodRoute(e *echo.Echo, h *FeedbackPeriodHandler, callerTeamID string) {
	e.DELETE("/teams/:teamId/periods/:periodId", func(c *echo.Context) error {
		c.Set("team_id", callerTeamID)
		return h.DeletePeriod(c)
	})
}

func setupGetActivePeriodRoute(e *echo.Echo, h *FeedbackPeriodHandler, callerTeamID string) {
	e.GET("/teams/:teamId/periods/active", func(c *echo.Context) error {
		c.Set("team_id", callerTeamID)
		return h.GetActivePeriod(c)
	})
}

// samplePeriod returns a FeedbackPeriod for use in tests.
func samplePeriod() *model.FeedbackPeriod {
	now := time.Now().Truncate(time.Second)
	return &model.FeedbackPeriod{
		ID:        "period-1",
		TeamID:    "team-1",
		Name:      "2026-H1",
		StartDate: now.Add(-24 * time.Hour),
		EndDate:   now.Add(24 * time.Hour),
		CreatedBy: "manager-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ---------------------------------------------------------------------------
// CreatePeriod tests
// ---------------------------------------------------------------------------

func TestCreatePeriod_Handler_Success(t *testing.T) {
	p := samplePeriod()
	mockSvc := &mockFeedbackPeriodService{
		CreatePeriodFn: func(_ context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			period.ID = p.ID
			period.CreatedAt = p.CreatedAt
			period.UpdatedAt = p.UpdatedAt
			return period, nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := fmt.Sprintf(`{"name":"2026-H1","start_date":"%s","end_date":"%s"}`,
		p.StartDate.Format(time.RFC3339),
		p.EndDate.Format(time.RFC3339),
	)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp feedbackPeriodResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "period-1", resp.ID)
	assert.Equal(t, "team-1", resp.TeamID)
	assert.Equal(t, "2026-H1", resp.Name)
	assert.Equal(t, "manager-1", resp.CreatedBy)
}

func TestCreatePeriod_Handler_Forbidden(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-other", "manager-1")

	body := `{"name":"2026-H1","start_date":"2026-01-01T00:00:00Z","end_date":"2026-06-30T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "forbidden")
}

func TestCreatePeriod_Handler_InvalidBody(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestCreatePeriod_Handler_MissingFields(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	// Missing end_date
	body := `{"name":"2026-H1","start_date":"2026-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "required")
}

func TestCreatePeriod_Handler_InvalidStartDateFormat(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := `{"name":"2026-H1","start_date":"not-a-date","end_date":"2026-06-30T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid start_date format")
}

func TestCreatePeriod_Handler_InvalidEndDateFormat(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := `{"name":"2026-H1","start_date":"2026-01-01T00:00:00Z","end_date":"not-a-date"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid end_date format")
}

func TestCreatePeriod_Handler_EndBeforeStart(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		CreatePeriodFn: func(_ context.Context, _ *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			return nil, service.ErrPeriodEndBeforeStart
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := `{"name":"2026-H1","start_date":"2026-06-30T00:00:00Z","end_date":"2026-01-01T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "end date must be after start date")
}

func TestCreatePeriod_Handler_NameConflict(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		CreatePeriodFn: func(_ context.Context, _ *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			return nil, service.ErrPeriodNameExists
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := `{"name":"2026-H1","start_date":"2026-01-01T00:00:00Z","end_date":"2026-06-30T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "already exists")
}

func TestCreatePeriod_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		CreatePeriodFn: func(_ context.Context, _ *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupCreatePeriodRoute(e, h, "team-1", "team-1", "manager-1")

	body := `{"name":"2026-H1","start_date":"2026-01-01T00:00:00Z","end_date":"2026-06-30T00:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/periods", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to create period")
}

// ---------------------------------------------------------------------------
// ListPeriods tests
// ---------------------------------------------------------------------------

func TestListPeriods_Handler_Success(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	periods := []*model.FeedbackPeriod{
		{ID: "p2", TeamID: "team-1", Name: "2026-H2", StartDate: now.AddDate(0, 6, 0), EndDate: now.AddDate(1, 0, 0), CreatedAt: now, UpdatedAt: now},
		{ID: "p1", TeamID: "team-1", Name: "2026-H1", StartDate: now, EndDate: now.AddDate(0, 6, 0), CreatedAt: now, UpdatedAt: now},
	}

	mockSvc := &mockFeedbackPeriodService{
		ListPeriodsForTeamFn: func(_ context.Context, teamID string) ([]*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return periods, nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupListPeriodsRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []feedbackPeriodResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	assert.Equal(t, "p2", resp[0].ID)
	assert.Equal(t, "2026-H2", resp[0].Name)
	assert.Equal(t, "p1", resp[1].ID)
	assert.Equal(t, "2026-H1", resp[1].Name)
}

func TestListPeriods_Handler_Empty(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return []*model.FeedbackPeriod{}, nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupListPeriodsRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []feedbackPeriodResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestListPeriods_Handler_Forbidden(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupListPeriodsRoute(e, h, "team-other")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestListPeriods_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		ListPeriodsForTeamFn: func(_ context.Context, _ string) ([]*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupListPeriodsRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to list periods")
}

// ---------------------------------------------------------------------------
// DeletePeriod tests
// ---------------------------------------------------------------------------

func TestDeletePeriod_Handler_Success(t *testing.T) {
	var capturedTeamID, capturedPeriodID string
	mockSvc := &mockFeedbackPeriodService{
		DeletePeriodFn: func(_ context.Context, teamID, periodID string) error {
			capturedTeamID = teamID
			capturedPeriodID = periodID
			return nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupDeletePeriodRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/periods/period-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "team-1", capturedTeamID)
	assert.Equal(t, "period-1", capturedPeriodID)
}

func TestDeletePeriod_Handler_Forbidden(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupDeletePeriodRoute(e, h, "team-other")

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/periods/period-1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeletePeriod_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		DeletePeriodFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("period not found")
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupDeletePeriodRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/periods/nonexistent", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to delete period")
}

// ---------------------------------------------------------------------------
// GetActivePeriod tests
// ---------------------------------------------------------------------------

func TestGetActivePeriod_Handler_Active(t *testing.T) {
	p := samplePeriod()
	mockSvc := &mockFeedbackPeriodService{
		GetActivePeriodForTeamFn: func(_ context.Context, teamID string) (*model.FeedbackPeriod, error) {
			assert.Equal(t, "team-1", teamID)
			return p, nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupGetActivePeriodRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods/active", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp feedbackPeriodResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "period-1", resp.ID)
	assert.Equal(t, "team-1", resp.TeamID)
	assert.Equal(t, "2026-H1", resp.Name)
}

func TestGetActivePeriod_Handler_None(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string) (*model.FeedbackPeriod, error) {
			return nil, nil
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupGetActivePeriodRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods/active", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "no active feedback period")
}

func TestGetActivePeriod_Handler_Forbidden(t *testing.T) {
	h := NewFeedbackPeriodHandler(&mockFeedbackPeriodService{})
	e := echo.New()
	setupGetActivePeriodRoute(e, h, "team-other")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods/active", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetActivePeriod_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackPeriodService{
		GetActivePeriodForTeamFn: func(_ context.Context, _ string) (*model.FeedbackPeriod, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackPeriodHandler(mockSvc)
	e := echo.New()
	setupGetActivePeriodRoute(e, h, "team-1")

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/periods/active", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to get active period")
}
