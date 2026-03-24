package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type FeedbackDraftHandler struct {
	DraftService FeedbackDraftService
}

func NewFeedbackDraftHandler(draftService FeedbackDraftService) *FeedbackDraftHandler {
	return &FeedbackDraftHandler{DraftService: draftService}
}

func (h *FeedbackDraftHandler) SaveDraft(c *echo.Context) error {
	loggedInUserID, ok := c.Get("user_id").(string)
	if !ok || loggedInUserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	revieweeID := c.Param("revieweeId")
	if revieweeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "revieweeId is required"})
	}

	var req saveDraftRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "named"
	}
	if visibility != "named" && visibility != "anonymous" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "visibility must be 'named' or 'anonymous'"})
	}

	draft := &model.FeedbackDraft{
		ReviewerID:         loggedInUserID,
		RevieweeID:         revieweeID,
		CommunicationScore: req.CommunicationScore,
		LeadershipScore:    req.LeadershipScore,
		TechnicalScore:     req.TechnicalScore,
		CollaborationScore: req.CollaborationScore,
		DeliveryScore:      req.DeliveryScore,
		StrengthsComment:   req.StrengthsComment,
		WeaknessesComment:  req.WeaknessesComment,
		Visibility:         visibility,
	}

	ctx := c.Request().Context()
	saved, err := h.DraftService.SaveDraft(ctx, draft)
	if err != nil {
		if errors.Is(err, service.ErrFeedbackAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrReviewerNotFound) || errors.Is(err, service.ErrRevieweeNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrNoActivePeriod) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save draft"})
	}

	return c.JSON(http.StatusOK, toDraftResponse(saved))
}

func (h *FeedbackDraftHandler) GetDraft(c *echo.Context) error {
	loggedInUserID, ok := c.Get("user_id").(string)
	if !ok || loggedInUserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	revieweeID := c.Param("revieweeId")
	if revieweeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "revieweeId is required"})
	}

	ctx := c.Request().Context()
	draft, err := h.DraftService.GetDraft(ctx, loggedInUserID, revieweeID)
	if err != nil {
		if errors.Is(err, service.ErrReviewerNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get draft"})
	}

	if draft == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
	}

	return c.JSON(http.StatusOK, toDraftResponse(draft))
}

func (h *FeedbackDraftHandler) ListDrafts(c *echo.Context) error {
	loggedInUserID, ok := c.Get("user_id").(string)
	if !ok || loggedInUserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	drafts, err := h.DraftService.GetMyDrafts(ctx, loggedInUserID)
	if err != nil {
		if errors.Is(err, service.ErrReviewerNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list drafts"})
	}

	resp := make([]draftResponse, 0, len(drafts))
	for _, d := range drafts {
		resp = append(resp, toDraftResponse(d))
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *FeedbackDraftHandler) DeleteDraft(c *echo.Context) error {
	loggedInUserID, ok := c.Get("user_id").(string)
	if !ok || loggedInUserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	revieweeID := c.Param("revieweeId")
	if revieweeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "revieweeId is required"})
	}

	ctx := c.Request().Context()
	if err := h.DraftService.DeleteDraft(ctx, loggedInUserID, revieweeID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete draft"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
