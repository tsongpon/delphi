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
