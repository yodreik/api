package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func New(rdb *redis.Client) *Redis {
	return &Redis{
		client: rdb,
	}
}

func (r *Redis) SetPasswordResetRequest(ctx context.Context, email string, token string) error {
	return r.client.Set(ctx, token, email, 24*time.Hour).Err()
}

func (r *Redis) GetPasswordResetEmailByToken(ctx context.Context, token string) (string, error) {
	return r.client.Get(ctx, token).Result()
}
