package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/gateway"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/internal/login"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/bus"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/httpserver"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/logging"
	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/storage"
)

func main() {
	cfg, err := config.Load("gateway")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := logging.New(cfg.AppName, cfg.ServiceName, cfg.Env)

	redisClient := storage.NewRedis(cfg.RedisAddr)
	defer redisClient.Close()

	nc, err := bus.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	secret := os.Getenv("LOGIN_JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret"
	}

	instanceID := os.Getenv("GATEWAY_INSTANCE_ID")
	if instanceID == "" {
		id, idErr := newInstanceID()
		if idErr != nil {
			log.Fatalf("generate gateway instance id: %v", idErr)
		}
		instanceID = id
	}

	parser := login.NewAuthenticator(secret, 24*time.Hour)
	sender := gateway.NewSender(instanceID, logger, redisClient, parser)

	sub, err := gateway.SubscribeSendToUser(nc, logger, sender)
	if err != nil {
		log.Fatalf("subscribe to gateway subject: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	mux := httpserver.NewMux()
	sender.Register(mux)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := httpserver.Run(ctx, logger, 8080, mux, cfg.ShutdownTimeout); err != nil {
		log.Fatalf("gateway service failed: %v", err)
	}
}

func newInstanceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32]), nil
}
