// Package template
package template

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bikky-kc013/notification-system/shared/common"
	"github.com/bikky-kc013/notification-system/shared/domain"
	"github.com/bikky-kc013/notification-system/shared/middleware"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/templates", h.Create)
	g.PUT("/templates/:id", h.Update)
	g.GET("/templates/:id", h.GetActive)
	g.GET("/templates/:id/versions/:version", h.GetVersion)
	g.GET("/templates", h.List)
	g.DELETE("/templates/:id", h.Deactivate)
}

func (h *Handler) Create(c *echo.Context) error {
	var req CreateTemplateRequest
	if err := middleware.ValidateAndBind(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Create(c.Request().Context(), req)
	if err != nil {
		h.log.Error("create template failed", zap.Error(err))
		return common.NewInternalError("create template failed", err)
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) Update(c *echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return common.NewBadRequestError("template id is required", nil)
	}

	var req UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return common.NewBadRequestError("invalid request body", err)
	}
	resp, err := h.svc.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return common.NewNotFoundError("template not found", err)
		}
		if errors.Is(err, domain.ErrConcurrentUpdate) {
			return common.NewConflictError("concurrent update detected, retry")
		}
		h.log.Error("update template failed", zap.Error(err))
		return common.NewInternalError("update template failed", err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetActive(c *echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return common.NewBadRequestError("template id is required", nil)
	}

	resp, err := h.svc.GetActive(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return common.NewNotFoundError("template not found", err)
		}
		h.log.Error("get template failed", zap.Error(err))
		return common.NewInternalError("get template failed", err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetVersion(c *echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return common.NewBadRequestError("template id is required", nil)
	}

	version, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		return common.NewBadRequestError("invalid version", err)
	}

	resp, err := h.svc.GetVersion(c.Request().Context(), id, version)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return common.NewNotFoundError("template not found", err)
		}
		h.log.Error("get template version failed", zap.Error(err))
		return common.NewInternalError("get template version failed", err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) List(c *echo.Context) error {
	limit := parseInt(c.QueryParam("limit"), 20)
	cursor := c.QueryParam("cursor")

	resp, err := h.svc.List(c.Request().Context(), limit, cursor)
	if err != nil {
		h.log.Error("list templates failed", zap.Error(err))
		return common.NewInternalError("list templates failed", err)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Deactivate(c *echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return common.NewBadRequestError("template id is required", nil)
	}

	if err := h.svc.Deactivate(c.Request().Context(), id); err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return common.NewNotFoundError("template not found", err)
		}
		h.log.Error("deactivate template failed", zap.Error(err))
		return common.NewInternalError("deactivate template failed", err)
	}

	return c.NoContent(http.StatusNoContent)
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}
