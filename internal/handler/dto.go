package handler

import (
	"time"

	"github.com/tsongpon/delphi/internal/model"
)

type registerUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Title    string `json:"title"`
}

type userResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Title     string `json:"title"`
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
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
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
	ReviewerID         string `json:"reviewer_id"`
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
