// Package server
package server

import (
	"context"

	"github.com/bikky-kc013/notification-system/pkg/config"
	"github.com/bikky-kc013/notification-system/pkg/database"
	"github.com/bikky-kc013/notification-system/pkg/middleware"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

type Server struct {
	Echo   *echo.Echo
	logger *zap.Logger
	cfg    *config.Config
	db     *database.DBConnection
}

func New(cfg *config.Config, logger *zap.Logger, db *database.DBConnection) *Server {
	e := echo.New()
	s := &Server{
		Echo:   e,
		logger: logger,
		db:     db,
		cfg:    cfg,
	}
	s.registerMiddlewares()
	return s
}

func (s *Server) registerMiddlewares() {
	s.Echo.Use(middleware.RequestIDMiddleware())
	s.Echo.Use(middleware.RequestLoggerMiddleware(s.logger))
	s.Echo.HTTPErrorHandler = middleware.NewErrorHandler(s.logger, s.cfg.Env == "production")
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server", zap.String("port", s.cfg.Port))
	sc := echo.StartConfig{
		Address:    s.cfg.Port,
		HideBanner: true,
		HidePort:   true,
	}
	return sc.Start(ctx, s.Echo)
}
