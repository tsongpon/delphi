package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type FeedbackHandler struct {
	FeedbackService FeedbackService
}

func NewFeedbackHandler(feedbackService FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{FeedbackService: feedbackService}
}

func (h *FeedbackHandler) CreateFeedback(c *echo.Context) error {
	loggedInUserID, ok := c.Get("user_id").(string)
	if !ok || loggedInUserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req createFeedbackRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.ReviewerID != loggedInUserID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "reviewer_id does not match logged in user"})
	}

	if req.Visibility != "named" && req.Visibility != "anonymous" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "visibility must be 'named' or 'anonymous'"})
	}

	feedback := &model.Feedback{
		ReviewerID:         req.ReviewerID,
		RevieweeID:         req.RevieweeID,
		CommunicationScore: req.CommunicationScore,
		LeadershipScore:    req.LeadershipScore,
		TechnicalScore:     req.TechnicalScore,
		CollaborationScore: req.CollaborationScore,
		DeliveryScore:      req.DeliveryScore,
		StrengthsComment:   req.StrengthsComment,
		WeaknessesComment:  req.WeaknessesComment,
		Visibility:         req.Visibility,
	}

	ctx := c.Request().Context()
	created, err := h.FeedbackService.CreateFeedback(ctx, feedback)
	if err != nil {
		if errors.Is(err, service.ErrFeedbackAlreadyExists) {
			return c.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, service.ErrReviewerNotFound) || errors.Is(err, service.ErrRevieweeNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create feedback"})
	}

	return c.JSON(http.StatusCreated, toFeedbackResponse(created))
}

const defaultFeedbacksLimit = 15

func (h *FeedbackHandler) GetMyFeedbacks(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	limit := defaultFeedbacksLimit
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		parsed, err := strconv.Atoi(limitParam)
		if err != nil || parsed <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		}
		limit = parsed
	}
	cursor := c.QueryParam("cursor")

	ctx := c.Request().Context()
	// Request one extra to detect if a next page exists
	feedbacks, err := h.FeedbackService.GetFeedbacksForUser(ctx, userID, limit+1, cursor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get feedbacks"})
	}

	var nextCursor string
	if len(feedbacks) > limit {
		feedbacks = feedbacks[:limit]
		lastCreatedAt := feedbacks[limit-1].CreatedAt
		nextCursor = base64.StdEncoding.EncodeToString([]byte(lastCreatedAt.Format(time.RFC3339Nano)))
	}

	data := make([]feedbackResponse, 0, len(feedbacks))
	for _, f := range feedbacks {
		data = append(data, toFeedbackResponse(f))
	}

	return c.JSON(http.StatusOK, paginatedFeedbackResponse{Data: data, NextCursor: nextCursor})
}

func (h *FeedbackHandler) GetTeamDashboard(c *echo.Context) error {
	teamID := c.Param("teamId")
	if teamID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing teamId"})
	}

	ctx := c.Request().Context()
	dashboard, err := h.FeedbackService.GetTeamDashboard(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get team dashboard"})
	}

	members := make([]memberDashboardEntry, 0, len(dashboard.Members))
	for _, m := range dashboard.Members {
		members = append(members, memberDashboardEntry{
			ID:            m.ID,
			Name:          m.Name,
			Title:         m.Title,
			Email:         m.Email,
			AvgScore:      m.AvgScore,
			FeedbackCount: m.FeedbackCount,
			Scores: memberScores{
				Communication: m.Communication,
				Leadership:    m.Leadership,
				Technical:     m.Technical,
				Collaboration: m.Collaboration,
				Delivery:      m.Delivery,
			},
		})
	}

	return c.JSON(http.StatusOK, teamDashboardResponse{
		TeamMembers:      dashboard.TeamMembers,
		AvgTeamScore:     dashboard.AvgTeamScore,
		TotalFeedbacks:   dashboard.TotalFeedbacks,
		FeedbackCoverage: dashboard.FeedbackCoverage,
		Members:          members,
	})
}

func (h *FeedbackHandler) GetMemberFeedbacks(c *echo.Context) error {
	teamID := c.Param("teamId")
	memberID := c.Param("memberId")
	if teamID == "" || memberID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing teamId or memberId"})
	}

	limit := defaultFeedbacksLimit
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		parsed, err := strconv.Atoi(limitParam)
		if err != nil || parsed <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		}
		limit = parsed
	}
	cursor := c.QueryParam("cursor")

	ctx := c.Request().Context()
	feedbacks, err := h.FeedbackService.GetFeedbacksForMember(ctx, teamID, memberID, limit+1, cursor)
	if err != nil {
		if errors.Is(err, service.ErrRevieweeNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "member not found"})
		}
		if errors.Is(err, service.ErrMemberNotInTeam) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "member does not belong to this team"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get member feedbacks"})
	}

	var nextCursor string
	if len(feedbacks) > limit {
		feedbacks = feedbacks[:limit]
		lastCreatedAt := feedbacks[limit-1].CreatedAt
		nextCursor = base64.StdEncoding.EncodeToString([]byte(lastCreatedAt.Format(time.RFC3339Nano)))
	}

	data := make([]feedbackResponse, 0, len(feedbacks))
	for _, f := range feedbacks {
		data = append(data, toFeedbackResponse(f))
	}

	return c.JSON(http.StatusOK, paginatedFeedbackResponse{Data: data, NextCursor: nextCursor})
}

func (h *FeedbackHandler) GetTeamFeedbacks(c *echo.Context) error {
	teamID := c.Param("teamId")
	if teamID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing teamId"})
	}

	ctx := c.Request().Context()
	feedbacks, err := h.FeedbackService.GetTeamFeedbacks(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get team feedbacks"})
	}

	response := make([]feedbackResponse, 0, len(feedbacks))
	for _, f := range feedbacks {
		response = append(response, toFeedbackResponse(f))
	}
	return c.JSON(http.StatusOK, response)
}

func (h *FeedbackHandler) GetMyGivenFeedbacks(c *echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	limit := defaultFeedbacksLimit
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		parsed, err := strconv.Atoi(limitParam)
		if err != nil || parsed <= 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		}
		limit = parsed
	}
	cursor := c.QueryParam("cursor")

	ctx := c.Request().Context()
	// Request one extra to detect if a next page exists
	feedbacks, err := h.FeedbackService.GetGivenFeedbacksForUser(ctx, userID, limit+1, cursor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get given feedbacks"})
	}

	var nextCursor string
	if len(feedbacks) > limit {
		feedbacks = feedbacks[:limit]
		lastCreatedAt := feedbacks[limit-1].CreatedAt
		nextCursor = base64.StdEncoding.EncodeToString([]byte(lastCreatedAt.Format(time.RFC3339Nano)))
	}

	data := make([]feedbackResponse, 0, len(feedbacks))
	for _, f := range feedbacks {
		data = append(data, toFeedbackResponse(f))
	}

	return c.JSON(http.StatusOK, paginatedFeedbackResponse{Data: data, NextCursor: nextCursor})
}
