package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const SendToUserSubject = "pcgb.gateway.send_to_user"

var ErrOffline = errors.New("offline")

type GatewayLookup interface {
	GatewayInstanceID(ctx context.Context, userID string) (string, error)
}

type Publisher interface {
	Publish(subject string, data []byte) error
}

type Service struct {
	lookup      GatewayLookup
	publisher   Publisher
	partitioned bool
}

func NewService(lookup GatewayLookup, publisher Publisher, partitioned bool) *Service {
	return &Service{lookup: lookup, publisher: publisher, partitioned: partitioned}
}

type Envelope struct {
	UserID            string          `json:"user_id"`
	GatewayInstanceID string          `json:"gateway_instance_id"`
	Message           json.RawMessage `json:"message"`
}

func (s *Service) Route(ctx context.Context, userID string, message json.RawMessage) (string, error) {
	gatewayInstanceID, err := s.lookup.GatewayInstanceID(ctx, userID)
	if err != nil {
		return "", err
	}

	envelope := Envelope{UserID: userID, GatewayInstanceID: gatewayInstanceID, Message: message}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}

	subject := SendToUserSubject
	if s.partitioned {
		subject = fmt.Sprintf("%s.%s", SendToUserSubject, gatewayInstanceID)
	}

	if err := s.publisher.Publish(subject, payload); err != nil {
		return "", fmt.Errorf("publish route event: %w", err)
	}

	return gatewayInstanceID, nil
}

type redisGetter interface {
	Get(ctx context.Context, key string) *redis.StringCmd
}

type RedisLookup struct {
	client redisGetter
}

func NewRedisLookup(client redisGetter) *RedisLookup {
	return &RedisLookup{client: client}
}

func userGatewayKey(userID string) string {
	return fmt.Sprintf("pcgb:user_gateway:%s", userID)
}

func (l *RedisLookup) GatewayInstanceID(ctx context.Context, userID string) (string, error) {
	gatewayInstanceID, err := l.client.Get(ctx, userGatewayKey(userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrOffline
		}
		return "", fmt.Errorf("redis lookup: %w", err)
	}
	if gatewayInstanceID == "" {
		return "", ErrOffline
	}
	return gatewayInstanceID, nil
}
