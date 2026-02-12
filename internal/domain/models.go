package domain

import "time"

type Application struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Role struct {
	AppID       string    `json:"app_id"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Permission struct {
	AppID       string    `json:"app_id"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserAppRoles struct {
	UserID    string    `json:"user_id"`
	AppID     string    `json:"app_id"`
	Roles     []string  `json:"roles"`
	UpdatedAt time.Time `json:"updated_at"`
}
