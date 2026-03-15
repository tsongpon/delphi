package handler

import (
	"time"

	"github.com/tsongpon/delphi/internal/model"
)

type registerUserRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Title       string `json:"title"`
	TeamName    string `json:"team_name"`
	InviteToken string `json:"invite_token"`
}

type userResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Title     string `json:"title"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func toUserResponse(user *model.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Title:     user.Title,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

type createFeedbackRequest struct {
	ReviewerID         string `json:"reviewer_id"`
	RevieweeID         string `json:"reviewee_id"`
	CommunicationScore int    `json:"communication_score"`
	LeadershipScore    int    `json:"leadership_score"`
	TechnicalScore     int    `json:"technical_score"`
	CollaborationScore int    `json:"collaboration_score"`
	DeliveryScore      int    `json:"delivery_score"`
	StrengthsComment   string `json:"strengths_comment"`
	WeaknessesComment  string `json:"weaknesses_comment"`
	Visibility         string `json:"visibility"`
}

type feedbackResponse struct {
	ID                 string `json:"id"`
	Period             string `json:"period"`
	RevieweeID         string `json:"reviewee_id"`
	ReviewerID         string `json:"reviewer_id,omitempty"`
	CommunicationScore int    `json:"communication_score"`
	LeadershipScore    int    `json:"leadership_score"`
	TechnicalScore     int    `json:"technical_score"`
	CollaborationScore int    `json:"collaboration_score"`
	DeliveryScore      int    `json:"delivery_score"`
	StrengthsComment   string `json:"strengths_comment"`
	WeaknessesComment  string `json:"weaknesses_comment"`
	Visibility         string `json:"visibility"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type generateResetLinkResponse struct {
	ResetLink string `json:"reset_link"`
	ExpiresAt string `json:"expires_at"`
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type paginatedFeedbackResponse struct {
	Data       []feedbackResponse `json:"data"`
	NextCursor string             `json:"next_cursor"`
}

type memberScores struct {
	Communication float64 `json:"communication"`
	Leadership    float64 `json:"leadership"`
	Technical     float64 `json:"technical"`
	Collaboration float64 `json:"collaboration"`
	Delivery      float64 `json:"delivery"`
}

type memberDashboardEntry struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Title         string       `json:"title"`
	Email         string       `json:"email"`
	AvgScore      float64      `json:"avg_score"`
	FeedbackCount int          `json:"feedback_count"`
	Scores        memberScores `json:"scores"`
}

type teamDashboardResponse struct {
	TeamMembers      int                    `json:"team_members"`
	AvgTeamScore     float64                `json:"avg_team_score"`
	TotalFeedbacks   int                    `json:"total_feedbacks"`
	FeedbackCoverage int                    `json:"feedback_coverage"`
	Members          []memberDashboardEntry `json:"members"`
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type createInviteLinkRequest struct {
	ExpiresInDays int `json:"expires_in_days"`
}

type inviteLinkResponse struct {
	ID         string `json:"id"`
	InviteLink string `json:"invite_link"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
	UsedCount  int    `json:"used_count"`
}

type listInviteLinksResponse struct {
	InviteLinks []inviteLinkResponse `json:"invite_links"`
}

type validateTokenResponse struct {
	Valid     bool   `json:"valid"`
	TeamID    string `json:"team_id,omitempty"`
	TeamName  string `json:"team_name,omitempty"`
	Role      string `json:"role,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type teamResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toTeamResponse(t *model.Team) teamResponse {
	return teamResponse{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

func toFeedbackResponse(f *model.Feedback) feedbackResponse {
	return feedbackResponse{
		ID:                 f.ID,
		Period:             f.Period,
		RevieweeID:         f.RevieweeID,
		ReviewerID:         f.ReviewerID,
		CommunicationScore: f.CommunicationScore,
		LeadershipScore:    f.LeadershipScore,
		TechnicalScore:     f.TechnicalScore,
		CollaborationScore: f.CollaborationScore,
		DeliveryScore:      f.DeliveryScore,
		StrengthsComment:   f.StrengthsComment,
		WeaknessesComment:  f.WeaknessesComment,
		Visibility:         f.Visibility,
		CreatedAt:          f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          f.UpdatedAt.Format(time.RFC3339),
	}
}
