package main

import (
	"context"

	"os"
	"os/signal"
	"syscall"

	"github.com/bikky-kc013/notification-system/pkg/config"
	"github.com/bikky-kc013/notification-system/pkg/database"
	"github.com/bikky-kc013/notification-system/pkg/logger"
	"github.com/bikky-kc013/notification-system/services/template-svc/internal/server"
	"github.com/bikky-kc013/notification-system/services/template-svc/internal/template"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	err = logger.Init(cfg.Env)
	if err != nil {
		os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := logger.Get()
	db, err := database.NewDBConnection(cfg)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	store := template.NewPGStore(database.NewDatabase(db.DB))
	svc := template.NewService(store)
	h := template.NewHandler(svc, logger)

	srv := server.New(cfg, logger, db)
	h.RegisterRoutes(srv.Echo.Group("/api/v1"))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		logger.Error("server error", zap.Error(err))
	}

	if err := db.Close(); err != nil {
		logger.Error("db close error", zap.Error(err))
	}

	_ = logger.Sync()
}
