package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/service"
)

func setupNotifyRoute(h *NotifyHandler) (*echo.Echo, string) {
	e := echo.New()
	route := "/admin/feedback-notify"
	e.POST(route, h.SendFeedbackDigest)
	return e, route
}

func postNotify(e *echo.Echo, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/feedback-notify", &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- All-users (no team_id) ---

func TestSendFeedbackDigest_Handler_NoBody_AllUsers(t *testing.T) {
	var capturedTeamID string
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, teamID string) (*service.NotifyResult, error) {
			capturedTeamID = teamID
			return &service.NotifyResult{Notified: 3, Skipped: 2}, nil
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	req := httptest.NewRequest(http.MethodPost, "/admin/feedback-notify", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", capturedTeamID)

	var body map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 3, body["notified"])
	assert.Equal(t, 2, body["skipped"])
}

func TestSendFeedbackDigest_Handler_EmptyTeamID_AllUsers(t *testing.T) {
	var capturedTeamID string
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, teamID string) (*service.NotifyResult, error) {
			capturedTeamID = teamID
			return &service.NotifyResult{Notified: 5, Skipped: 0}, nil
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	rec := postNotify(e, map[string]string{"team_id": ""})

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", capturedTeamID)

	var body map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 5, body["notified"])
}

// --- Team-scoped ---

func TestSendFeedbackDigest_Handler_TeamID_PassedToService(t *testing.T) {
	var capturedTeamID string
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, teamID string) (*service.NotifyResult, error) {
			capturedTeamID = teamID
			return &service.NotifyResult{Notified: 2, Skipped: 1}, nil
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	rec := postNotify(e, map[string]string{"team_id": "team-abc"})

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "team-abc", capturedTeamID)

	var body map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 2, body["notified"])
	assert.Equal(t, 1, body["skipped"])
}

func TestSendFeedbackDigest_Handler_TeamID_AllSkipped(t *testing.T) {
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, _ string) (*service.NotifyResult, error) {
			return &service.NotifyResult{Notified: 0, Skipped: 4}, nil
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	rec := postNotify(e, map[string]string{"team_id": "team-xyz"})

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]int
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 0, body["notified"])
	assert.Equal(t, 4, body["skipped"])
}

// --- Error handling ---

func TestSendFeedbackDigest_Handler_ServiceError(t *testing.T) {
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, _ string) (*service.NotifyResult, error) {
			return nil, fmt.Errorf("db unavailable")
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	rec := postNotify(e, nil)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "failed to send feedback digest", body["error"])
}

func TestSendFeedbackDigest_Handler_TeamServiceError(t *testing.T) {
	svc := &mockNotifyService{
		SendFeedbackDigestFn: func(_ context.Context, teamID string) (*service.NotifyResult, error) {
			if teamID == "bad-team" {
				return nil, fmt.Errorf("team not found")
			}
			return &service.NotifyResult{}, nil
		},
	}
	h := NewNotifyHandler(svc)
	e, _ := setupNotifyRoute(h)

	rec := postNotify(e, map[string]string{"team_id": "bad-team"})

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "failed to send feedback digest", body["error"])
}
