package userpreference

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bikky-kc013/notification-system/shared/database"
)

type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unexpected type for Stringslice: %T", src)
	}
	return json.Unmarshal(b, s)
}

type UserPreferenceModel struct {
	UserPreferenceID string          `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ClientID         string          `gorm:"type:varchar(100);not null;uniqueIndex:idx_client_external"`
	ExternalUserID   string          `gorm:"type:varchar(255);not null;uniqueIndex:idx_client_external"`
	Channels         map[string]bool `gorm:"type:jsonb;not null"`
	Types            map[string]bool `gorm:"type:jsonb;not null"`
	DoNotDisturb     bool            `gorm:"default:false;not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func toModel(up UserPreference) UserPreferenceModel {
	return UserPreferenceModel{
		UserPreferenceID: up.ID,
		ClientID:         up.ClientID,
		ExternalUserID:   up.ExternalUserID,
		Channels:         up.Channels,
		Types:            up.Types,
		DoNotDisturb:     up.DoNotDisturb,
		CreatedAt:        up.CreatedAt,
		UpdatedAt:        up.UpdatedAt,
	}
}

func toDomain(m UserPreferenceModel) UserPreference {
	return UserPreference{
		ID:             m.UserPreferenceID,
		ClientID:       m.ClientID,
		ExternalUserID: m.ExternalUserID,
		Channels:       m.Channels,
		Types:          m.Types,
		DoNotDisturb:   m.DoNotDisturb,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type PGStore struct {
	db *database.Database
}

func NewPGStore(db *database.Database) *PGStore {
	return &PGStore{db: db}
}

func (s *PGStore) Create(ctx context.Context, up UserPreference) error {
	m := toModel(up)
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create user notification preference: %w", err)
	}
	return nil
}
