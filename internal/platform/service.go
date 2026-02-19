package platform

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

// RunBootService starts a baseline service with shared dependencies and health endpoints.
func RunBootService(serviceName string) error {
	cfg, err := config.Load(serviceName)
	if err != nil {
		return err
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)
	logger.Info().Msg("loading shared dependencies")

	db, err := storage.NewPostgres(cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer db.Close()

	redisClient := storage.NewRedis(cfg.RedisAddr)
	defer redisClient.Close()

	natsConn, err := bus.Connect(cfg.NATSURL)
	if err != nil {
		return err
	}
	defer natsConn.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := httpserver.NewMux(cfg.ServiceName)
	return httpserver.Run(ctx, logger, cfg.HTTPPort, mux, cfg.ShutdownTimeout)
}
