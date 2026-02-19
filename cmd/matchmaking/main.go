package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/matchmaking"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("matchmaking")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)
	port := 8084
	if cfg.HTTPPort != 8080 {
		port = cfg.HTTPPort
	}

	redisClient := storage.NewRedis(cfg.RedisAddr)
	defer redisClient.Close()

	nc, err := bus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	secret := os.Getenv("MATCHMAKING_JWT_SECRET")
	if secret == "" {
		secret = os.Getenv("LOGIN_JWT_SECRET")
	}
	if secret == "" {
		secret = "local-dev-secret"
	}

	queue := matchmaking.NewRedisQueue(redisClient)
	svc := matchmaking.NewService(queue, nc)
	auth := login.NewAuthenticator(secret, 24*time.Hour)
	handler := matchmaking.NewHandler(svc, auth)

	mux := httpserver.NewMux()
	handler.Register(mux)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go svc.Run(ctx, 2*time.Second)

	if err := httpserver.Run(ctx, logger, port, mux, cfg.ShutdownTimeout); err != nil {
		log.Fatalf("matchmaking service failed: %v", err)
	}
}
