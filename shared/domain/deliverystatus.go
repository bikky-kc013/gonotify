package domain

import "time"

type DeliveryStatus struct {
	ID                string        `json:"id"`
	NotificationID    string        `json:"notification_id"`
	Channel           Channel       `json:"channel"`
	Provider          string        `json:"provider"`
	ProviderMessageID string        `json:"provider_message_id"`
	State             DeliveryState `json:"state"`
	FailureClass      FailureClass  `json:"failure_class,omitempty"`
	FailureReason     string        `json:"failure_reason,omitempty"`
	AttemptNumber     int           `json:"attempt_number"`
	CreatedAt         time.Time     `json:"created_at"`
}
