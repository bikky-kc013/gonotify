package domain

import "time"

type Notification struct {
	ID             string            `json:"id"`
	ClientID       string            `json:"client_id"`
	ExternalUserID string            `json:"external_user_id"`
	TemplateID     string            `json:"template_id"`
	Channel        Channel           `json:"channel"`
	Priority       Priority          `json:"priority"`
	Type           NotificationType  `json:"type"`
	Status         NotificationStatus `json:"status"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	Variables      map[string]string `json:"variables"`
	ScheduledAt    *time.Time        `json:"scheduled_at,omitempty"`
	RetryCount     int               `json:"retry_count"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
