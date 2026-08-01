// Package pagination
package pagination

import (
	"math"

	"github.com/bikky-kc013/notification-system/shared/common"
	"github.com/labstack/echo/v5"
)

const (
	DefaultLimit  = 20
	MaxLimit      = 100
	DefaultOffset = 0
)

// Params represents pagination parameters
type Params struct {
	Limit  int `form:"limit" json:"limit"`
	Offset int `form:"offset" json:"offset"`
}

// ParseParams extracts and validates pagination parameters from request
func ParseParams(c *echo.Context) Params {
	params := Params{
		Limit:  DefaultLimit,
		Offset: DefaultOffset,
	}
	// Bind query parameters
	if err := c.Bind(&params); err != nil {
		// If binding fails, use defaults
		return params
	}

	// Validate and sanitize limit
	if params.Limit <= 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}

	// Validate and sanitize offset
	if params.Offset < 0 {
		params.Offset = DefaultOffset
	}

	return params
}

// BuildMeta creates pagination metadata for responses
func BuildMeta(limit, offset int, total int64) *common.Meta {
	meta := &common.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  total,
	}
	// Calculate total pages
	if limit > 0 {
		meta.TotalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	return meta
}

// HasMore checks if there are more items available
func HasMore(offset, limit int, total int64) bool {
	return int64(offset+limit) < total
}

// GetCurrentPage calculates the current page number (1-indexed)
func GetCurrentPage(offset, limit int) int {
	if limit <= 0 {
		return 1
	}
	return (offset / limit) + 1
}
