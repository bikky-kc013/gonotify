package template

import (
	"time"

	_ "github.com/bikky-kc013/notification-system/pkg/domain"
)

type Template struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Name      string    `json:"name"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Variables []string  `json:"variables"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
