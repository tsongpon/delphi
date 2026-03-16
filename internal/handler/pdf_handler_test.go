package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

func setupExportPDFRoute(e *echo.Echo, h *FeedbackHandler, userID, name string) {
	e.GET("/me/feedbacks/export", func(c *echo.Context) error {
		c.Set("user_id", userID)
		c.Set("name", name)
		return h.ExportMyFeedbacksPDF(c)
	})
}

// ---------------------------------------------------------------------------
// splitPeriod unit tests
// ---------------------------------------------------------------------------

func TestSplitPeriod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1-2026", "Q1 2026"},
		{"2-2025", "Q2 2025"},
		{"3-2024", "Q3 2024"},
		{"4-2023", "Q4 2023"},
		{"invalid", ""},
		{"", ""},
		{"1-", ""},
		{"-2026", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitPeriod(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// reviewerLabel unit tests
// ---------------------------------------------------------------------------

func TestReviewerLabel(t *testing.T) {
	t.Run("anonymous visibility returns Anonymous", func(t *testing.T) {
		entry := &service.FeedbackExportEntry{
			Feedback:     &model.Feedback{Visibility: "anonymous"},
			ReviewerName: "John Doe",
		}
		assert.Equal(t, "Anonymous", reviewerLabel(entry))
	})

	t.Run("named visibility with reviewer name returns name", func(t *testing.T) {
		entry := &service.FeedbackExportEntry{
			Feedback:     &model.Feedback{Visibility: "named"},
			ReviewerName: "Jane Smith",
		}
		assert.Equal(t, "Jane Smith", reviewerLabel(entry))
	})

	t.Run("named visibility with empty reviewer name returns Named Reviewer", func(t *testing.T) {
		entry := &service.FeedbackExportEntry{
			Feedback:     &model.Feedback{Visibility: "named"},
			ReviewerName: "",
		}
		assert.Equal(t, "Named Reviewer", reviewerLabel(entry))
	})
}

// ---------------------------------------------------------------------------
// ExportMyFeedbacksPDF handler tests
// ---------------------------------------------------------------------------

func TestExportMyFeedbacksPDF_Unauthorized(t *testing.T) {
	mockSvc := &mockFeedbackService{}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()

	// Route without setting user_id
	e.GET("/me/feedbacks/export", func(c *echo.Context) error {
		return h.ExportMyFeedbacksPDF(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestExportMyFeedbacksPDF_ServiceError(t *testing.T) {
	mockSvc := &mockFeedbackService{
		ExportFeedbacksForUserFn: func(_ context.Context, _ string) ([]*service.FeedbackExportEntry, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()
	setupExportPDFRoute(e, h, "user-123", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to generate report")
}

func TestExportMyFeedbacksPDF_Success_Empty(t *testing.T) {
	mockSvc := &mockFeedbackService{
		ExportFeedbacksForUserFn: func(_ context.Context, userID string) ([]*service.FeedbackExportEntry, error) {
			assert.Equal(t, "user-123", userID)
			return []*service.FeedbackExportEntry{}, nil
		},
	}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()
	setupExportPDFRoute(e, h, "user-123", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="feedback-report.pdf"`, rec.Header().Get("Content-Disposition"))
	assert.Greater(t, rec.Body.Len(), 0)
}

func TestExportMyFeedbacksPDF_Success_WithFeedbacks(t *testing.T) {
	now := time.Now()
	entries := []*service.FeedbackExportEntry{
		{
			Feedback: &model.Feedback{
				ID:                 "fb-1",
				Period:             "1-2026",
				ReviewerID:         "reviewer-1",
				RevieweeID:         "user-123",
				CommunicationScore: 5,
				LeadershipScore:    4,
				TechnicalScore:     5,
				CollaborationScore: 4,
				DeliveryScore:      3,
				StrengthsComment:   "Great communicator",
				WeaknessesComment:  "Could improve delivery",
				Visibility:         "named",
				CreatedAt:          now,
			},
			ReviewerName: "Alice",
		},
		{
			Feedback: &model.Feedback{
				ID:                 "fb-2",
				Period:             "1-2026",
				ReviewerID:         "reviewer-2",
				RevieweeID:         "user-123",
				CommunicationScore: 4,
				LeadershipScore:    3,
				TechnicalScore:     4,
				CollaborationScore: 5,
				DeliveryScore:      4,
				StrengthsComment:   "",
				WeaknessesComment:  "",
				Visibility:         "anonymous",
				CreatedAt:          now,
			},
			ReviewerName: "",
		},
	}

	mockSvc := &mockFeedbackService{
		ExportFeedbacksForUserFn: func(_ context.Context, _ string) ([]*service.FeedbackExportEntry, error) {
			return entries, nil
		},
	}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()
	setupExportPDFRoute(e, h, "user-123", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="feedback-report.pdf"`, rec.Header().Get("Content-Disposition"))
	// PDF should have non-trivial content
	assert.Greater(t, rec.Body.Len(), 1000)
}

func TestExportMyFeedbacksPDF_Success_NoUserName(t *testing.T) {
	mockSvc := &mockFeedbackService{
		ExportFeedbacksForUserFn: func(_ context.Context, _ string) ([]*service.FeedbackExportEntry, error) {
			return []*service.FeedbackExportEntry{}, nil
		},
	}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()
	// Route without name set (empty string)
	setupExportPDFRoute(e, h, "user-123", "")

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
}

func TestExportMyFeedbacksPDF_Success_ManyFeedbacks(t *testing.T) {
	now := time.Now()
	// Create enough feedbacks to trigger page break
	entries := make([]*service.FeedbackExportEntry, 20)
	for i := range entries {
		entries[i] = &service.FeedbackExportEntry{
			Feedback: &model.Feedback{
				ID:                 fmt.Sprintf("fb-%d", i),
				Period:             fmt.Sprintf("%d-2025", (i%4)+1),
				ReviewerID:         fmt.Sprintf("reviewer-%d", i),
				RevieweeID:         "user-123",
				CommunicationScore: 4,
				LeadershipScore:    3,
				TechnicalScore:     4,
				CollaborationScore: 5,
				DeliveryScore:      4,
				StrengthsComment:   "Good work overall with some strong points to highlight",
				WeaknessesComment:  "Could improve in certain areas like communication and delivery",
				Visibility:         "named",
				CreatedAt:          now,
			},
			ReviewerName: fmt.Sprintf("Reviewer %d", i),
		}
	}

	mockSvc := &mockFeedbackService{
		ExportFeedbacksForUserFn: func(_ context.Context, _ string) ([]*service.FeedbackExportEntry, error) {
			return entries, nil
		},
	}
	h := NewFeedbackHandler(mockSvc)
	e := echo.New()
	setupExportPDFRoute(e, h, "user-123", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/me/feedbacks/export", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Greater(t, rec.Body.Len(), 1000)
}
