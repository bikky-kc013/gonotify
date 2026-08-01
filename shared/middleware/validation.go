package middleware

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"sort"

	"github.com/bikky-kc013/notification-system/shared/common"
	"github.com/bikky-kc013/notification-system/shared/logger"
	"github.com/bikky-kc013/notification-system/shared/validation"
	"github.com/labstack/echo/v5"
)

// ValidateRequest is a generic middleware that validates request body against a struct
// Usage: router.POST("/endpoint", middleware.ValidateRequest(&validation.CreateTemplateRequest{}), handler)
// middleware.go
func ValidateRequest(requestType interface{}) echo.MiddlewareFunc {
	t := reflect.TypeOf(requestType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := reflect.New(t).Interface()

			if err := c.Bind(req); err != nil {
				return common.ErrorResponse(c, http.StatusBadRequest, "invalid request format")
			}

			if err := validation.ValidateStruct(req); err != nil {
				return respondValidation(c, err)
			}

			c.Set("validatedRequest", req)
			return next(c)
		}
	}
}

// respondValidation is the single place that converts a validation error into the standard Response
func respondValidation(c *echo.Context, err error) error {
	fields := toFieldErrors(err)

	return c.JSON(http.StatusBadRequest, common.Response{
		Success: false,
		Error: &common.ErrorInfo{
			Code:      http.StatusBadRequest,
			ErrorCode: common.ErrCodeValidation,
			Message:   "validation failed",
			Fields:    fields,
		},
		CorrelationID: logger.CorrelationIDFromContext(c.Request().Context()),
	})
}

// in middleware package
func toFieldErrors(err error) []common.FieldError {
	valErr, ok := err.(*validation.ValidationError)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(valErr.Errors))
	for field := range valErr.Errors {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	out := make([]common.FieldError, 0, len(fields))
	for _, field := range fields {
		out = append(out, common.FieldError{Field: field, Message: valErr.Errors[field]})
	}
	return out
}

// ValidateJSON binds and validates a request body against the provided struct.
// This is a helper function to be used within handlers.
func ValidateJSON(c *echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return err
	}
	return validation.ValidateStruct(req)
}

// ValidateQuery binds and validates query parameters against a struct.
// Echo's default binder populates struct fields from query params via `query` tags.
func ValidateQuery(c *echo.Context, req interface{}) error {
	if err := (&echo.DefaultBinder{}).Bind(c, req); err != nil {
		return err
	}
	return validation.ValidateStruct(req)
}

// ValidateURI binds and validates URI (path) parameters against a struct.
// Echo's default binder populates struct fields from path params via `param` tags.
func ValidateURI(c *echo.Context, req interface{}) error {
	if err := (&echo.DefaultBinder{}).Bind(c, req); err != nil {
		return err
	}
	return validation.ValidateStruct(req)
}

// RespondWithValidationError sends a standardized validation error response
func RespondWithValidationError(c *echo.Context, err error) error {
	return respondValidation(c, err)
}

// ValidateAndBind validates and binds the request body to the provided struct.
// Returns nil on success. On failure it writes the error response and returns
// the error so the caller can simply `return` it from the handler.
func ValidateAndBind(c *echo.Context, req interface{}) error {
	if err := ValidateJSON(c, req); err != nil {
		return RespondWithValidationError(c, err)
	}
	return nil
}

// ValidateAndBindQuery validates and binds query parameters to the provided struct.
func ValidateAndBindQuery(c *echo.Context, req interface{}) error {
	if err := ValidateQuery(c, req); err != nil {
		return RespondWithValidationError(c, err)
	}
	return nil
}

// GetValidatedRequest retrieves the validated request from context.
// This is used when ValidateRequest middleware is applied.
func GetValidatedRequest(c *echo.Context) (interface{}, bool) {
	req := c.Get("validatedRequest")
	return req, req != nil
}

// ValidateContentType ensures the request has the expected content type
func ValidateContentType(contentType string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if c.Request().Header.Get(echo.HeaderContentType) != contentType {
				return c.JSON(http.StatusUnsupportedMediaType, map[string]any{
					"error":    "Unsupported content type",
					"expected": contentType,
					"received": c.Request().Header.Get(echo.HeaderContentType),
				})
			}
			return next(c)
		}
	}
}

// ValidateJSONContentType ensures the request has application/json content type
func ValidateJSONContentType() echo.MiddlewareFunc {
	return ValidateContentType("application/json")
}

// MaxBodySize limits the request body size
func MaxBodySize(maxSize int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			if req.Body == nil {
				return next(c)
			}

			// Wrap the body with a size limiter and read it.
			req.Body = http.MaxBytesReader(c.Response(), req.Body, maxSize)
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return c.JSON(http.StatusRequestEntityTooLarge, map[string]any{
					"error":          "Request body too large",
					"max_size_bytes": maxSize,
				})
			}

			// Restore the body so downstream handlers can read it.
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			return next(c)
		}
	}
}
