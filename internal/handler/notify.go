package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// feedbackNotifyRequest is the optional body for POST /admin/feedback-notify.
// Omitting team_id (or sending an empty string) notifies all users.
type feedbackNotifyRequest struct {
	TeamID string `json:"team_id"`
}

type NotifyHandler struct {
	NotifyService NotifyService
}

func NewNotifyHandler(notifyService NotifyService) *NotifyHandler {
	return &NotifyHandler{NotifyService: notifyService}
}

// SendFeedbackDigest triggers a feedback digest email for users who received
// new feedback since yesterday. When team_id is provided in the request body
// only members of that team are notified. Responds with the number of emails
// sent and skipped.
func (h *NotifyHandler) SendFeedbackDigest(c *echo.Context) error {
	var req feedbackNotifyRequest
	// Body is optional; ignore bind errors (empty body is valid).
	_ = c.Bind(&req)

	ctx := c.Request().Context()
	result, err := h.NotifyService.SendFeedbackDigest(ctx, req.TeamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send feedback digest"})
	}
	return c.JSON(http.StatusOK, map[string]int{"notified": result.Notified, "skipped": result.Skipped})
}
