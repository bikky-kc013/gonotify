package domain

import "errors"

var (
	ErrTemplateNotFound     = errors.New("template not found")
	ErrTemplateInactive     = errors.New("template is not active")
	ErrMissingVariable      = errors.New("missing required variable")
	ErrInvalidChannel       = errors.New("invalid channel")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrPreferencesNotFound  = errors.New("preferences not found")
	ErrAllChannelsBlocked   = errors.New("all requested channels blocked by preferences")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrDuplicateDelivery    = errors.New("delivery already recorded, idempotent skip")
	ErrRateLimited          = errors.New("rate limited, retry later")
	ErrConcurrentUpdate     = errors.New("concurrent update detected, retry")
)
