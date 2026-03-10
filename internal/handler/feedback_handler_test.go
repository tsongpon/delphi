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

// helper: registers POST /feedbacks with user_id injected into context
func setupFeedbackRoute(e *echo.Echo, h *FeedbackHandler, loggedInUserID string) {
	e.POST("/feedbacks", func(c *echo.Context) error {
		c.Set("user_id", loggedInUserID)
		return h.CreateFeedback(c)
	})
}

func TestCreateFeedback_Handler_Success(t *testing.T) {
	now := time.Now()

	mockSvc := &mockFeedbackService{
		CreateFeedbackFn: func(_ context.Context, feedback *model.Feedback) (*model.Feedback, error) {
			feedback.ID = "feedback-uuid"
			feedback.Period = "1-2026"
			feedback.CreatedAt = now
			feedback.UpdatedAt = now
			return feedback, nil
		},
	}

	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "reviewer-123",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"leadership_score": 4,
		"technical_score": 5,
		"collaboration_score": 4,
		"delivery_score": 3,
		"strengths_comment": "Great work",
		"weaknesses_comment": "Could improve",
		"visibility": "named"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp feedbackResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "feedback-uuid", resp.ID)
	assert.Equal(t, "1-2026", resp.Period)
	assert.Equal(t, "reviewer-123", resp.ReviewerID)
	assert.Equal(t, "reviewee-123", resp.RevieweeID)
	assert.Equal(t, 5, resp.CommunicationScore)
	assert.Equal(t, 4, resp.LeadershipScore)
	assert.Equal(t, 5, resp.TechnicalScore)
	assert.Equal(t, 4, resp.CollaborationScore)
	assert.Equal(t, 3, resp.DeliveryScore)
	assert.Equal(t, "Great work", resp.StrengthsComment)
	assert.Equal(t, "Could improve", resp.WeaknessesComment)
	assert.Equal(t, "named", resp.Visibility)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
}

func TestCreateFeedback_Handler_InvalidBody(t *testing.T) {
	mockSvc := &mockFeedbackService{}
	h := NewFeedbackHandler(mockSvc)

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader("not-json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestCreateFeedback_Handler_InvalidVisibility(t *testing.T) {
	mockSvc := &mockFeedbackService{}
	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "reviewer-123",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"visibility": "public"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "visibility must be 'named' or 'anonymous'")
}

func TestCreateFeedback_Handler_ReviewerIDMismatch(t *testing.T) {
	mockSvc := &mockFeedbackService{}
	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "someone-else",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"visibility": "named"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "reviewer_id does not match logged in user")
}

func TestCreateFeedback_Handler_Duplicate(t *testing.T) {
	mockSvc := &mockFeedbackService{
		CreateFeedbackFn: func(_ context.Context, _ *model.Feedback) (*model.Feedback, error) {
			return nil, service.ErrFeedbackAlreadyExists
		},
	}

	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "reviewer-123",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"visibility": "named"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "feedback already exists for this period")
}

func TestCreateFeedback_Handler_ReviewerNotFound(t *testing.T) {
	mockSvc := &mockFeedbackService{
		CreateFeedbackFn: func(_ context.Context, _ *model.Feedback) (*model.Feedback, error) {
			return nil, service.ErrReviewerNotFound
		},
	}

	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "nonexistent",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"visibility": "named"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "nonexistent")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "reviewer not found")
}

func TestCreateFeedback_Handler_RevieweeNotFound(t *testing.T) {
	mockSvc := &mockFeedbackService{
		CreateFeedbackFn: func(_ context.Context, _ *model.Feedback) (*model.Feedback, error) {
			return nil, service.ErrRevieweeNotFound
		},
	}

	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "reviewer-123",
		"reviewee_id": "nonexistent",
		"communication_score": 5,
		"visibility": "named"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "reviewee not found")
}

// ---------------------------------------------------------------------------
// GetMyFeedbacks handler tests
// ---------------------------------------------------------------------------

func TestGetMyFeedbacks_Handler_Success(t *testing.T) {
	now := time.Now()

	mockSvc := &mockFeedbackService{
		GetFeedbacksForUserFn: func(_ context.Context, userID string) ([]*model.Feedback, error) {
			assert.Equal(t, "user-123", userID)
			return []*model.Feedback{
				{ID: "fb-1", Period: "1-2026", ReviewerID: "reviewer-1", RevieweeID: "user-123", CommunicationScore: 5, Visibility: "named", CreatedAt: now, UpdatedAt: now},
				{ID: "fb-2", Period: "1-2026", ReviewerID: "reviewer-2", RevieweeID: "user-123", CommunicationScore: 4, Visibility: "anonymous", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}

	h := NewFeedbackHandler(mockSvc)

	e := echo.New()
	e.GET("/me/feedbacks", func(c *echo.Context) error {
		c.Set("user_id", "user-123")
		return h.GetMyFeedbacks(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []feedbackResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, 2)
	assert.Equal(t, "fb-1", resp[0].ID)
	assert.Equal(t, "fb-2", resp[1].ID)
}

func TestGetMyFeedbacks_Handler_Empty(t *testing.T) {
	mockSvc := &mockFeedbackService{
		GetFeedbacksForUserFn: func(_ context.Context, _ string) ([]*model.Feedback, error) {
			return []*model.Feedback{}, nil
		},
	}

	h := NewFeedbackHandler(mockSvc)

	e := echo.New()
	e.GET("/me/feedbacks", func(c *echo.Context) error {
		c.Set("user_id", "user-123")
		return h.GetMyFeedbacks(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []feedbackResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp)
}

func TestGetMyFeedbacks_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackService{
		GetFeedbacksForUserFn: func(_ context.Context, _ string) ([]*model.Feedback, error) {
			return nil, fmt.Errorf("firestore error")
		},
	}

	h := NewFeedbackHandler(mockSvc)

	e := echo.New()
	e.GET("/me/feedbacks", func(c *echo.Context) error {
		c.Set("user_id", "user-123")
		return h.GetMyFeedbacks(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to get feedbacks")
}

func TestCreateFeedback_Handler_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackService{
		CreateFeedbackFn: func(_ context.Context, _ *model.Feedback) (*model.Feedback, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}

	h := NewFeedbackHandler(mockSvc)

	body := `{
		"reviewer_id": "reviewer-123",
		"reviewee_id": "reviewee-123",
		"communication_score": 5,
		"visibility": "anonymous"
	}`

	e := echo.New()
	setupFeedbackRoute(e, h, "reviewer-123")

	req := httptest.NewRequest(http.MethodPost, "/feedbacks", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to create feedback")
}
