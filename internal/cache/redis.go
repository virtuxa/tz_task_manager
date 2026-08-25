package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func OpenRedis(ctx context.Context, address string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: address})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return client, nil
}
