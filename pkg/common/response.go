// Package common
package common

import (
	"net/http"

	"github.com/bikky-kc013/notification-system/pkg/logger"
	"github.com/labstack/echo/v5"
)

// Response represents a standard API response
type Response struct {
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         *ErrorInfo  `json:"error,omitempty"`
	Meta          *Meta       `json:"meta,omitempty"`
	CorrelationID string      `json:"correlation_id,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code      int          `json:"code"`
	ErrorCode string       `json:"error_code,omitempty"`
	Message   string       `json:"message"`
	Fields    []FieldError `json:"fields,omitempty"`
}
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Meta contains metadata for paginated responses
type Meta struct {
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	Total      int64       `json:"total,omitempty"`
	TotalPages int         `json:"total_pages,omitempty"`
	Stats      interface{} `json:"stats,omitempty"`
}

// SuccessResponse sends a successful response (backward compatibility)
func SuccessResponse(c *echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessResponseWithStatus sends a successful response with custom status code
func SuccessResponseWithStatus(c *echo.Context, statusCode int, data interface{}, message string) error {
	return c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessResponseWithMeta sends a successful response with metadata
func SuccessResponseWithMeta(c *echo.Context, data interface{}, meta *Meta) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// SuccessResponseWithMetaAndStatus sends a successful response with metadata and status
func SuccessResponseWithMetaAndStatus(c *echo.Context, statusCode int, data interface{}, meta *Meta, message string) error {
	return c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// CreatedResponse sends a created response
func CreatedResponse(c *echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    statusCode,
			Message: message,
		},
		CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
	})
}

// NoRouteHandler returns an echo.HandlerFunc for unregistered routes (404)
func NoRouteHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error: &ErrorInfo{
				Code:      http.StatusNotFound,
				ErrorCode: ErrCodeNotFound,
				Message:   "route not found",
			},
			CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
		})
	}
}

// NoMethodHandler returns an echo.HandlerFunc for unsupported methods (405)
func NoMethodHandler() echo.HandlerFunc {
	return func(c *echo.Context) error {
		return c.JSON(http.StatusMethodNotAllowed, Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    http.StatusMethodNotAllowed,
				Message: "method not allowed",
			},
			CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
		})
	}
}

// AppErrorResponse sends an AppError response
func AppErrorResponse(c *echo.Context, err *AppError) error {
	return c.JSON(err.Code, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      err.Code,
			ErrorCode: err.ErrorCode,
			Message:   err.Message,
		},
		CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
	})
}

// ValidationErrorResponse sends a 400 response with field-level errors
func ValidationErrorResponse(c *echo.Context, fields []FieldError) error {
	return c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:      http.StatusBadRequest,
			ErrorCode: ErrCodeValidation,
			Message:   "validation failed",
			Fields:    fields,
		},
		CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
	})
}
