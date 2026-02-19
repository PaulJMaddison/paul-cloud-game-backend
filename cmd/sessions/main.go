package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/contracts"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/sessions"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/observability"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("sessions")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)

	otelShutdown, err := observability.InitOTEL(context.Background(), cfg, logger)
	if err != nil {
		log.Fatalf("init otel: %v", err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()
	port := 8083
	if cfg.HTTPPort != 8080 {
		port = cfg.HTTPPort
	}

	db, err := storage.NewPostgres(cfg.PostgresURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("close postgres connection: %v", closeErr)
		}
	}()

	nc, err := bus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	secret := os.Getenv("LOGIN_JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret"
	}

	auth := login.NewAuthenticator(secret, 24*time.Hour)
	repo := sessions.NewPostgresRepository(db)
	redisClient := storage.NewRedis(cfg.RedisAddr)
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Printf("close redis client: %v", closeErr)
		}
	}()

	svc := sessions.NewService(repo, auth, nc, redisClient)
	handler := sessions.NewHandler(svc)

	if _, err := nc.Subscribe(contracts.SubjectMatchmakingMatch, svc.HandleMatchedEvent); err != nil {
		log.Fatalf("subscribe matched events: %v", err)
	}

	mux := httpserver.NewMux(cfg.ServiceName)
	handler.Register(mux)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := httpserver.Run(ctx, logger, port, mux, cfg.ShutdownTimeout); err != nil {
		log.Fatalf("sessions service failed: %v", err)
	}
}
