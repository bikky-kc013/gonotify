package userpreference

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, req CreateUserPreferenceRequest) (*UserPreferenceResponse, error) {
	userPreference := UserPreference{
		ID:             uuid.New().String(),
		ClientID:       req.ClientID,
		ExternalUserID: req.ExternalUserID,
		Channels:       req.Channels,
		Types:          req.Types,
		DoNotDisturb:   req.DoNotDisturb,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.store.Create(ctx, userPreference); err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return toResponse(userPreference), nil

}

func toResponse(up UserPreference) *UserPreferenceResponse {
	return &UserPreferenceResponse{
		ID:             up.ID,
		ExternalUserID: up.ExternalUserID,
		Channels:       up.Channels,
		Types:          up.Types,
		DoNotDisturb:   up.DoNotDisturb,
		ClientID:       up.ClientID,
		CreatedAt:      up.CreatedAt,
	}
}
