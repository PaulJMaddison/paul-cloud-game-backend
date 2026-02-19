package matchmaking

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const defaultQueueKey = "pcgb:mm:queue"

type Queue interface {
	Enqueue(ctx context.Context, userID string) error
	DequeuePair(ctx context.Context) ([]string, error)
}

type RedisQueue struct {
	client *redis.Client
	key    string
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client, key: defaultQueueKey}
}

func (q *RedisQueue) Enqueue(ctx context.Context, userID string) error {
	return q.client.RPush(ctx, q.key, userID).Err()
}

func (q *RedisQueue) DequeuePair(ctx context.Context) ([]string, error) {
	ids, err := q.client.LPopCount(ctx, q.key, 2).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(ids) == 1 {
		if pushErr := q.client.LPush(ctx, q.key, ids[0]).Err(); pushErr != nil {
			return nil, pushErr
		}
		return nil, nil
	}
	if len(ids) < 2 {
		return nil, nil
	}
	return ids, nil
}
