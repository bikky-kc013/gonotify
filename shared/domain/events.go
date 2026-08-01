package domain

import "time"

type NotificationCreatedEvent struct {
	NotificationID string `json:"notification_id"`
	Channel        Channel `json:"channel"`
	Priority       Priority `json:"priority"`
	Payload        []byte  `json:"payload"`
	RetryCount     int     `json:"retry_count"`
}

type PreferenceUpdatedEvent struct {
	ClientID       string `json:"client_id"`
	ExternalUserID string `json:"external_user_id"`
}

type WebhookDeliveryEvent struct {
	Provider          string        `json:"provider"`
	ProviderMessageID string        `json:"provider_message_id"`
	Status            DeliveryState `json:"status"`
	RawPayload        []byte        `json:"raw_payload"`
	ReceivedAt        time.Time     `json:"received_at"`
}
