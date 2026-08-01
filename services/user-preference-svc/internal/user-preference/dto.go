package userpreference

import "time"

type CreateUserPreferenceRequest struct {
	ExternalUserID string          `json:"external_user_id" validate:"required,min=3,max=255"`
	Channels       map[string]bool `json:"channels" validate:"omitempty,dive,oneof=EMAIL SMS PUSH"`
	Types          map[string]bool `json:"types" validate:"omitempty,dive,min=1,max=50"`
	DoNotDisturb   bool            `json:"do_not_disturb,omitempty"`
	ClientID       string          `json:"client_id" validate:"required,min=3,max=255"`
}

type UserPreferenceResponse struct {
	ID             string          `json:"id"`
	ExternalUserID string          `json:"external_user_id"`
	Channels       map[string]bool `json:"channels"`
	Types          map[string]bool `json:"types"`
	DoNotDisturb   bool            `json:"do_not_disturb"`
	ClientID       string          `json:"client_id"`
	CreatedAt      time.Time       `json:"created_at"`
}
