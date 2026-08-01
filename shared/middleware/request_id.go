// Package middleware
package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bikky-kc013/notification-system/shared/logger"
	"github.com/labstack/echo/v5"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			id := req.Header.Get("X-Request-ID")
			if id == "" {
				id = generateRequestID()
			}
			c.Set(string(RequestIDKey), id)
			response := c.Response().(*echo.Response)
			response.Header().Set("X-Request-ID", id)

			// Bridge: propagate request ID into Go's context.Context so
			// logger.CorrelationIDFromContext and error responses can find it.
			ctx := logger.ContextWithCorrelationID(req.Context(), id)
			c.SetRequest(req.WithContext(ctx))

			return next(c)
		}
	}
}

func GetRequestID(c *echo.Context) string {
	if id, ok := c.Get(string(RequestIDKey)).(string); ok {
		return id
	}
	return ""
}
