package model

import "time"

type InviteLink struct {
	ID        string // UUID — also the jti claim in the JWT
	Token     string // Full signed JWT
	TeamID    string
	TeamName  string
	CreatedBy string // user ID
	Role      string // "member"
	ExpiresAt time.Time
	UsedCount int
	CreatedAt time.Time
}
