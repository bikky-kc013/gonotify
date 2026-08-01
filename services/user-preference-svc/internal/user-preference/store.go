package userpreference

import "context"

type Store interface {
	Create(ctx context.Context, u UserPreference) error
	// List(ctx context.Context, limit int, cursor int) ([]UserPreference, string, error)
	// Update(ctx context.Context, u UserPreference) error
}
