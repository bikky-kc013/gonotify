package middleware

import (
	"errors"
	"net/http"

	"github.com/bikky-kc013/notification-system/pkg/common"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

func NewErrorHandler(log *zap.Logger, isProduction bool) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {

		resp, ok := c.Response().(*echo.Response)
		if ok && resp.Committed {
			return
		}
		status := echo.StatusCode(err)
		if status == 0 {
			status = http.StatusInternalServerError
		}

		if status == http.StatusNotFound {
			log.Warn("route not found",
				zap.String("path", c.Request().URL.Path),
				zap.String("method", c.Request().Method),
			)
			_ = c.JSON(http.StatusNotFound, common.NewRouteNotFoundError("route not found", err))
			return
		}

		var appErr *common.AppError
		if errors.As(err, &appErr) {
			log.Error("app error", zap.Error(err))
			_ = c.JSON(status, appErr)
			return
		}

		log.Error("unhandled error", zap.Int("status", status), zap.Error(err))
		_ = c.JSON(status, common.NewInternalError("internal server error", err))
	}
}
