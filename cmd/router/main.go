package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/router"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/observability"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("router")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)

	otelShutdown, err := observability.InitOTEL(context.Background(), cfg, logger)
	if err != nil {
		log.Fatalf("init otel: %v", err)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	port := 8082
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

	lookup := router.NewRedisLookup(redisClient)
	routeService := router.NewService(lookup, nc, partitioningEnabled())
	handler := router.NewHandler(routeService)

	mux := httpserver.NewMux(cfg.ServiceName)
	handler.Register(mux)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := httpserver.Run(ctx, logger, port, mux, cfg.ShutdownTimeout); err != nil {
		log.Fatalf("router service failed: %v", err)
	}
}

func partitioningEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("ROUTER_PARTITIONED_SUBJECTS")))
	return value == "1" || value == "true" || value == "yes"
}
