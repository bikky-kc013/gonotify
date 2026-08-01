package userpreference

import (
	"net/http"

	"github.com/bikky-kc013/notification-system/shared/common"
	"github.com/bikky-kc013/notification-system/shared/middleware"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

type Handler struct {
	Svc *Service
	Log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{Svc: svc, Log: log}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("user-preferences", h.Create)
}

func (h *Handler) Create(ctx *echo.Context) error {
	var req CreateUserPreferenceRequest
	if err := middleware.ValidateAndBind(ctx, &req); err != nil {
		return err
	}
	resp, err := h.Svc.Create(ctx.Request().Context(), req)
	if err != nil {
		h.Log.Error("create user preference failed", zap.Error(err))
		return common.NewInternalError("create user preference failed", err)
	}
	return ctx.JSON(http.StatusCreated, resp)
}
