package middleware

import (
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

func RequestLoggerMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()
			path := c.Path()
			uri := c.Request().RequestURI

			err := next(c)

			resp := c.Response().(*echo.Response)
			status := resp.Status

			fields := []zap.Field{
				zap.String("method", c.Request().Method),
				zap.String("path", path),
				zap.String("uri", uri),
				zap.Int("status", status),
				zap.Duration("latency", time.Since(start)),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.String("request_id", GetRequestID(c)),
			}
			if err != nil {
				fields = append(fields, zap.Error(err))
			}

			if isHealthCheck(path) {
				logger.Debug("HTTP Request", fields...)
				return err
			}

			logger.Info("HTTP Request", fields...)
			return err
		}
	}
}

func isHealthCheck(path string) bool {
	return strings.HasSuffix(path, "/healthz") || strings.HasSuffix(path, "/readyz")
}
