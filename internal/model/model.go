package model

import "time"

type Team struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID        string
	Name      string
	Email     string
	Password  string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	TeamID    string
}

type PasswordResetToken struct {
	TokenHash string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Feedback struct {
	ID                 string
	Period             string
	RevieweeID         string
	ReviewerID         string
	CommunicationScore int
	LeadershipScore    int
	TechnicalScore     int
	CollaborationScore int
	DeliveryScore      int
	StrengthsComment   string
	WeaknessesComment  string
	Visibility         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
