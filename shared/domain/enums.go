package domain

type Channel string

const (
	ChannelEmail Channel = "EMAIL"
	ChannelSMS   Channel = "SMS"
	ChannelPush  Channel = "PUSH"
	ChannelInApp Channel = "INAPP"
)

func (c Channel) Valid() bool {
	switch c {
	case ChannelEmail, ChannelSMS, ChannelPush, ChannelInApp:
		return true
	}
	return false
}

type Priority string

const (
	PriorityCritical Priority = "CRITICAL"
	PriorityHigh     Priority = "HIGH"
	PriorityNormal   Priority = "NORMAL"
	PriorityLow      Priority = "LOW"
)

type NotificationType string

const (
	TypeTransactional NotificationType = "TRANSACTIONAL"
	TypePromotional   NotificationType = "PROMOTIONAL"
	TypeAlert         NotificationType = "ALERT"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "PENDING"
	StatusScheduled NotificationStatus = "SCHEDULED"
	StatusSent      NotificationStatus = "SENT"
	StatusDelivered NotificationStatus = "DELIVERED"
	StatusFailed    NotificationStatus = "FAILED"
	StatusCancelled NotificationStatus = "CANCELLED"
)

var validTransitions = map[NotificationStatus][]NotificationStatus{
	StatusPending:   {StatusScheduled, StatusSent, StatusFailed, StatusCancelled},
	StatusScheduled: {StatusPending, StatusCancelled},
	StatusSent:      {StatusDelivered, StatusFailed},
	StatusDelivered: {},
	StatusFailed:    {},
	StatusCancelled: {},
}

func (s NotificationStatus) CanTransitionTo(next NotificationStatus) bool {
	for _, allowed := range validTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

type DeliveryState string

const (
	DeliverySent      DeliveryState = "SENT"
	DeliveryDelivered DeliveryState = "DELIVERED"
	DeliveryFailed    DeliveryState = "FAILED"
	DeliveryBounced   DeliveryState = "BOUNCED"
	DeliveryOpened    DeliveryState = "OPENED"
	DeliveryClicked   DeliveryState = "CLICKED"
)

type FailureClass string

const (
	FailureTemporary FailureClass = "TEMPORARY"
	FailurePermanent FailureClass = "PERMANENT"
)
