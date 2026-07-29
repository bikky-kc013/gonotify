package template

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bikky-kc013/notification-system/pkg/database"
	"github.com/bikky-kc013/notification-system/pkg/domain"
	"gorm.io/gorm"
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
		return fmt.Errorf("unexpected type for StringSlice: %T", src)
	}
	return json.Unmarshal(b, s)
}

type templateModel struct {
	TemplateID string      `gorm:"column:template_id;primaryKey"`
	Version    int         `gorm:"column:version;primaryKey"`
	Name       string      `gorm:"column:name"`
	Channel    string      `gorm:"column:channel"`
	Subject    string      `gorm:"column:subject"`
	Body       string      `gorm:"column:body"`
	Variables  StringSlice `gorm:"column:variables;type:jsonb;serializer:json"`
	IsActive   bool        `gorm:"column:is_active"`
	CreatedAt  time.Time   `gorm:"column:created_at"`
}

func (templateModel) TableName() string { return "templates" }

func toModel(t Template) templateModel {
	return templateModel{
		TemplateID: t.ID,
		Version:    t.Version,
		Name:       t.Name,
		Channel:    t.Channel,
		Subject:    t.Subject,
		Body:       t.Body,
		Variables:  StringSlice(t.Variables),
		IsActive:   t.IsActive,
		CreatedAt:  t.CreatedAt,
	}
}

func toDomain(m templateModel) Template {
	return Template{
		ID:        m.TemplateID,
		Version:   m.Version,
		Name:      m.Name,
		Channel:   m.Channel,
		Subject:   m.Subject,
		Body:      m.Body,
		Variables: []string(m.Variables),
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
	}
}

type PGStore struct {
	db *database.Database
}

func NewPGStore(db *database.Database) *PGStore {
	return &PGStore{db: db}
}

func (s *PGStore) Create(ctx context.Context, t Template) error {
	m := toModel(t)
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	return nil
}

func (s *PGStore) GetActive(ctx context.Context, id string) (Template, error) {
	var m templateModel
	err := s.db.WithContext(ctx).
		Where("template_id = ? AND is_active = ?", id, true).
		First(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Template{}, domain.ErrTemplateNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get active template: %w", err)
	}
	return toDomain(m), nil
}

func (s *PGStore) GetVersion(ctx context.Context, id string, version int) (Template, error) {
	var m templateModel
	err := s.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", id, version).
		First(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Template{}, domain.ErrTemplateNotFound
	}
	if err != nil {
		return Template{}, fmt.Errorf("get template version: %w", err)
	}
	return toDomain(m), nil
}

func (s *PGStore) CreateNewVersion(ctx context.Context, next Template) error {
	return s.db.WithTxn(ctx, func(tx *database.Database) error {
		res := tx.Model(&templateModel{}).
			Where("template_id = ? AND is_active = ?", next.ID, true).
			Update("is_active", false)

		if res.Error != nil {
			return fmt.Errorf("deactivate current version: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return domain.ErrTemplateNotFound
		}

		m := toModel(next)
		if err := tx.Create(&m).Error; err != nil {
			if isUniqueViolation(err) {
				return domain.ErrConcurrentUpdate
			}
			return fmt.Errorf("insert new version: %w", err)
		}
		return nil
	})
}

func (s *PGStore) Deactivate(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).
		Model(&templateModel{}).
		Where("template_id = ? AND is_active = ?", id, true).
		Update("is_active", false)

	if res.Error != nil {
		return fmt.Errorf("deactivate template: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrTemplateNotFound
	}
	return nil
}

func (s *PGStore) List(ctx context.Context, limit int, cursor string) ([]Template, string, error) {
	q := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at DESC").
		Limit(limit)

	if cursor != "" {
		q = q.Where("created_at < (SELECT created_at FROM templates WHERE template_id = ?)", cursor)
	}

	var rows []templateModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, "", fmt.Errorf("list templates: %w", err)
	}

	out := make([]Template, len(rows))
	for i, m := range rows {
		out[i] = toDomain(m)
	}

	var next string
	if len(out) == limit {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
