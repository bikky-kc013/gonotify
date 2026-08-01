package domain

import (
	"fmt"
	"time"
)

type Template struct {
	ID        string
	Version   int
	Name      string
	Channel   Channel
	Subject   string
	Body      string
	Variables []string
	IsActive  bool
	CreatedAt time.Time
}

func (t Template) ValidateVariables(provided map[string]string) error {
	for _, v := range t.Variables {
		if _, ok := provided[v]; !ok {
			return fmt.Errorf("%w: %s", ErrMissingVariable, v)
		}
	}
	return nil
}

func (t Template) NextVersion(subject, body string, variables []string) Template {
	return Template{
		ID:        t.ID,
		Version:   t.Version + 1,
		Name:      t.Name,
		Channel:   t.Channel,
		Subject:   subject,
		Body:      body,
		Variables: variables,
		IsActive:  true,
	}
}
