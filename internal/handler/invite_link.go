package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
)

type InviteLinkHandler struct {
	InviteLinkService InviteLinkService
}

func NewInviteLinkHandler(inviteLinkService InviteLinkService) *InviteLinkHandler {
	return &InviteLinkHandler{InviteLinkService: inviteLinkService}
}

func toInviteLinkResponse(link *model.InviteLink, inviteURL string) inviteLinkResponse {
	return inviteLinkResponse{
		ID:         link.ID,
		InviteLink: inviteURL,
		ExpiresAt:  link.ExpiresAt.Format(time.RFC3339),
		CreatedAt:  link.CreatedAt.Format(time.RFC3339),
		UsedCount:  link.UsedCount,
	}
}

// CreateInviteLink handles POST /teams/:teamId/invite-links
func (h *InviteLinkHandler) CreateInviteLink(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	callerID, _ := c.Get("user_id").(string)

	var req createInviteLinkRequest
	// Ignore bind error — body is optional, defaults applied in service
	_ = c.Bind(&req)

	ctx := c.Request().Context()
	link, inviteURL, err := h.InviteLinkService.CreateInviteLink(ctx, teamID, callerID, req.ExpiresInDays)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create invite link"})
	}

	resp := struct {
		InviteLink string `json:"invite_link"`
		ExpiresAt  string `json:"expires_at"`
	}{
		InviteLink: inviteURL,
		ExpiresAt:  link.ExpiresAt.Format(time.RFC3339),
	}
	return c.JSON(http.StatusCreated, resp)
}

// ListInviteLinks handles GET /teams/:teamId/invite-links
func (h *InviteLinkHandler) ListInviteLinks(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	ctx := c.Request().Context()
	links, err := h.InviteLinkService.ListLinks(ctx, teamID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list invite links"})
	}

	items := make([]inviteLinkResponse, 0, len(links))
	for _, l := range links {
		// Reconstruct the invite URL from the stored token
		items = append(items, inviteLinkResponse{
			ID:         l.ID,
			InviteLink: l.Token, // Token is the full signed JWT; frontend uses it as query param
			ExpiresAt:  l.ExpiresAt.Format(time.RFC3339),
			CreatedAt:  l.CreatedAt.Format(time.RFC3339),
			UsedCount:  l.UsedCount,
		})
	}
	return c.JSON(http.StatusOK, listInviteLinksResponse{InviteLinks: items})
}

// RevokeInviteLink handles DELETE /teams/:teamId/invite-links/:linkId
// It hard-deletes the invite link record from Firestore.
func (h *InviteLinkHandler) RevokeInviteLink(c *echo.Context) error {
	teamID := c.Param("teamId")
	callerTeamID, _ := c.Get("team_id").(string)
	if callerTeamID != teamID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
	}

	linkID := c.Param("linkId")
	ctx := c.Request().Context()

	if err := h.InviteLinkService.DeleteLink(ctx, teamID, linkID); err != nil {
		if errors.Is(err, service.ErrInviteLinkNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "invite link not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete invite link"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ValidateInviteToken handles GET /invite-links/validate?token=<token>
func (h *InviteLinkHandler) ValidateInviteToken(c *echo.Context) error {
	rawToken := c.QueryParam("token")
	if rawToken == "" {
		return c.JSON(http.StatusOK, validateTokenResponse{Valid: false, Reason: "invalid"})
	}

	ctx := c.Request().Context()
	link, err := h.InviteLinkService.ValidateToken(ctx, rawToken)
	if err != nil {
		reason := "invalid"
		if errors.Is(err, service.ErrInviteLinkExpired) {
			reason = "expired"
		}
		return c.JSON(http.StatusOK, validateTokenResponse{Valid: false, Reason: reason})
	}

	return c.JSON(http.StatusOK, validateTokenResponse{
		Valid:     true,
		TeamID:    link.TeamID,
		TeamName:  link.TeamName,
		Role:      link.Role,
		ExpiresAt: link.ExpiresAt.Format(time.RFC3339),
	})
}
