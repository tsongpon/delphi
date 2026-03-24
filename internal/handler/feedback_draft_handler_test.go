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

// ── Route Setup Helpers ────────────────────────────────────────────────────

func setupSaveDraftRoute(e *echo.Echo, h *FeedbackDraftHandler, loggedInUserID string) {
	e.PUT("/me/drafts/:revieweeId", func(c *echo.Context) error {
		c.Set("user_id", loggedInUserID)
		return h.SaveDraft(c)
	})
}

func setupGetDraftRoute(e *echo.Echo, h *FeedbackDraftHandler, loggedInUserID string) {
	e.GET("/me/drafts/:revieweeId", func(c *echo.Context) error {
		c.Set("user_id", loggedInUserID)
		return h.GetDraft(c)
	})
}

func setupListDraftsRoute(e *echo.Echo, h *FeedbackDraftHandler, loggedInUserID string) {
	e.GET("/me/drafts", func(c *echo.Context) error {
		c.Set("user_id", loggedInUserID)
		return h.ListDrafts(c)
	})
}

func setupDeleteDraftRoute(e *echo.Echo, h *FeedbackDraftHandler, loggedInUserID string) {
	e.DELETE("/me/drafts/:revieweeId", func(c *echo.Context) error {
		c.Set("user_id", loggedInUserID)
		return h.DeleteDraft(c)
	})
}

// sampleDraft returns a FeedbackDraft with all fields populated for testing.
func sampleDraft(reviewerID, revieweeID string) *model.FeedbackDraft {
	now := time.Now()
	return &model.FeedbackDraft{
		ID:                 "draft-uuid-1",
		Period:             "2026-H1",
		ReviewerID:         reviewerID,
		RevieweeID:         revieweeID,
		CommunicationScore: 4,
		LeadershipScore:    3,
		TechnicalScore:     5,
		CollaborationScore: 4,
		DeliveryScore:      3,
		StrengthsComment:   "Good teamwork",
		WeaknessesComment:  "Could improve docs",
		Visibility:         "named",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// ── SaveDraft handler tests ────────────────────────────────────────────────

func TestSaveDraft_Handler_Success(t *testing.T) {
	draft := sampleDraft("reviewer-123", "reviewee-456")

	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, d *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			assert.Equal(t, "reviewer-123", d.ReviewerID)
			assert.Equal(t, "reviewee-456", d.RevieweeID)
			assert.Equal(t, 4, d.CommunicationScore)
			assert.Equal(t, "named", d.Visibility)
			return draft, nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	body := `{
		"communication_score": 4,
		"leadership_score": 3,
		"technical_score": 5,
		"collaboration_score": 4,
		"delivery_score": 3,
		"strengths_comment": "Good teamwork",
		"weaknesses_comment": "Could improve docs",
		"visibility": "named"
	}`

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp draftResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "draft-uuid-1", resp.ID)
	assert.Equal(t, "2026-H1", resp.Period)
	assert.Equal(t, "reviewer-123", resp.ReviewerID)
	assert.Equal(t, "reviewee-456", resp.RevieweeID)
	assert.Equal(t, 4, resp.CommunicationScore)
	assert.Equal(t, "Good teamwork", resp.StrengthsComment)
	assert.Equal(t, "named", resp.Visibility)
}

func TestSaveDraft_Handler_DefaultsVisibilityToNamed(t *testing.T) {
	var capturedVisibility string

	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, d *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			capturedVisibility = d.Visibility
			return sampleDraft("reviewer-123", "reviewee-456"), nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	// no visibility field in body
	body := `{"communication_score": 3}`
	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "named", capturedVisibility)
}

func TestSaveDraft_Handler_InvalidVisibility(t *testing.T) {
	h := NewFeedbackDraftHandler(&mockFeedbackDraftService{})
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	body := `{"visibility": "public"}`
	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSaveDraft_Handler_Unauthorized(t *testing.T) {
	h := NewFeedbackDraftHandler(&mockFeedbackDraftService{})
	e := echo.New()
	// Register route without injecting user_id
	e.PUT("/me/drafts/:revieweeId", h.SaveDraft)

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSaveDraft_Handler_NoActivePeriod(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, _ *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			return nil, service.ErrNoActivePeriod
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(`{"visibility":"named"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSaveDraft_Handler_FeedbackAlreadyExists(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, _ *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			return nil, service.ErrFeedbackAlreadyExists
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(`{"visibility":"named"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestSaveDraft_Handler_RevieweeNotFound(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, _ *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			return nil, service.ErrRevieweeNotFound
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(`{"visibility":"named"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSaveDraft_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		SaveDraftFn: func(_ context.Context, _ *model.FeedbackDraft) (*model.FeedbackDraft, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupSaveDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPut, "/me/drafts/reviewee-456", strings.NewReader(`{"visibility":"named"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── GetDraft handler tests ─────────────────────────────────────────────────

func TestGetDraft_Handler_Success(t *testing.T) {
	draft := sampleDraft("reviewer-123", "reviewee-456")

	mockSvc := &mockFeedbackDraftService{
		GetDraftFn: func(_ context.Context, reviewerID, revieweeID string) (*model.FeedbackDraft, error) {
			assert.Equal(t, "reviewer-123", reviewerID)
			assert.Equal(t, "reviewee-456", revieweeID)
			return draft, nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupGetDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp draftResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "draft-uuid-1", resp.ID)
	assert.Equal(t, "2026-H1", resp.Period)
}

func TestGetDraft_Handler_NotFound(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		GetDraftFn: func(_ context.Context, _, _ string) (*model.FeedbackDraft, error) {
			return nil, nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupGetDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetDraft_Handler_Unauthorized(t *testing.T) {
	h := NewFeedbackDraftHandler(&mockFeedbackDraftService{})
	e := echo.New()
	e.GET("/me/drafts/:revieweeId", h.GetDraft)

	req := httptest.NewRequest(http.MethodGet, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetDraft_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		GetDraftFn: func(_ context.Context, _, _ string) (*model.FeedbackDraft, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupGetDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── ListDrafts handler tests ───────────────────────────────────────────────

func TestListDrafts_Handler_Success(t *testing.T) {
	now := time.Now()
	drafts := []*model.FeedbackDraft{
		{ID: "d1", ReviewerID: "reviewer-123", RevieweeID: "r1", Period: "2026-H1", Visibility: "named", CreatedAt: now, UpdatedAt: now},
		{ID: "d2", ReviewerID: "reviewer-123", RevieweeID: "r2", Period: "2026-H1", Visibility: "anonymous", CreatedAt: now, UpdatedAt: now},
	}

	mockSvc := &mockFeedbackDraftService{
		GetMyDraftsFn: func(_ context.Context, reviewerID string) ([]*model.FeedbackDraft, error) {
			assert.Equal(t, "reviewer-123", reviewerID)
			return drafts, nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupListDraftsRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []draftResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, "d1", resp[0].ID)
	assert.Equal(t, "d2", resp[1].ID)
}

func TestListDrafts_Handler_EmptyList(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		GetMyDraftsFn: func(_ context.Context, _ string) ([]*model.FeedbackDraft, error) {
			return []*model.FeedbackDraft{}, nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupListDraftsRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []draftResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp)
}

func TestListDrafts_Handler_Unauthorized(t *testing.T) {
	h := NewFeedbackDraftHandler(&mockFeedbackDraftService{})
	e := echo.New()
	e.GET("/me/drafts", h.ListDrafts)

	req := httptest.NewRequest(http.MethodGet, "/me/drafts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListDrafts_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		GetMyDraftsFn: func(_ context.Context, _ string) ([]*model.FeedbackDraft, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupListDraftsRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodGet, "/me/drafts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ── DeleteDraft handler tests ──────────────────────────────────────────────

func TestDeleteDraft_Handler_Success(t *testing.T) {
	var deletedReviewerID, deletedRevieweeID string

	mockSvc := &mockFeedbackDraftService{
		DeleteDraftFn: func(_ context.Context, reviewerID, revieweeID string) error {
			deletedReviewerID = reviewerID
			deletedRevieweeID = revieweeID
			return nil
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupDeleteDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodDelete, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "reviewer-123", deletedReviewerID)
	assert.Equal(t, "reviewee-456", deletedRevieweeID)
}

func TestDeleteDraft_Handler_Unauthorized(t *testing.T) {
	h := NewFeedbackDraftHandler(&mockFeedbackDraftService{})
	e := echo.New()
	e.DELETE("/me/drafts/:revieweeId", h.DeleteDraft)

	req := httptest.NewRequest(http.MethodDelete, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestDeleteDraft_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackDraftService{
		DeleteDraftFn: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackDraftHandler(mockSvc)
	e := echo.New()
	setupDeleteDraftRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodDelete, "/me/drafts/reviewee-456", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
