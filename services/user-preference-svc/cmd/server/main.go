package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bikky-kc013/notification-system/services/user-preference-svc/internal/server"
	userpreference "github.com/bikky-kc013/notification-system/services/user-preference-svc/internal/user-preference"
	"github.com/bikky-kc013/notification-system/shared/config"
	"github.com/bikky-kc013/notification-system/shared/database"
	"github.com/bikky-kc013/notification-system/shared/logger"
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
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	store := userpreference.NewPGStore(database.NewDatabase(db.DB))
	service := userpreference.NewService(store)
	handler := userpreference.NewHandler(service, logger)
	server := server.New(cfg, logger, db)
	handler.RegisterRoutes(server.Echo.Group("/api/v1"))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Start(ctx); err != nil {
		logger.Error("server error", zap.Error(err))
	}
	if err := db.Close(); err != nil {
		logger.Error("db close error", zap.Error(err))
	}
	_ = logger.Sync()
}
