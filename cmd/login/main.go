package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/observability"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("login")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)

	otelShutdown, err := observability.InitOTEL(context.Background(), cfg, logger)
	if err != nil {
		log.Fatalf("init otel: %v", err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()
	port := 8081
	if cfg.HTTPPort != 8080 {
		port = cfg.HTTPPort
	}

	db, err := storage.NewPostgres(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	nc, err := bus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	secret := os.Getenv("LOGIN_JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret"
	}

	repo := login.NewPostgresRepository(db)
	auth := login.NewAuthenticator(secret, 24*time.Hour)
	svc := login.NewService(repo, auth, nc)
	handler := login.NewHandler(svc)

	mux := httpserver.NewMux(cfg.ServiceName)
	handler.Register(mux)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := httpserver.Run(ctx, logger, port, mux, cfg.ShutdownTimeout); err != nil {
		log.Fatalf("login service failed: %v", err)
	}
}
