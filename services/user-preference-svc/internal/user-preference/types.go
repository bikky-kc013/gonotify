package userpreference

import "time"

type UserPreference struct {
	ID             string          `json:"id"`
	ClientID       string          `json:"client_id"`
	ExternalUserID string          `json:"external_user_id"`
	Channels       map[string]bool `json:"channel"`
	Types          map[string]bool `json:"type"`
	DoNotDisturb   bool            `json:"do_not_disturb"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
