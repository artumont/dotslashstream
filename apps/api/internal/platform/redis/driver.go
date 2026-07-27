package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/artumont/dotslashstream/internal/platform"
	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	rdb *redis.Client
}

func New(redisAddr string) *RedisDriver {
	return &RedisDriver{
		rdb: redis.NewClient(&redis.Options{Addr: redisAddr}),
	}
}

func (r *RedisDriver) Close() error {
	return r.rdb.Close()
}

func (r *RedisDriver) PublishEvent(ctx context.Context, stream string, event platform.Event) (string, error) {
	bytes, err := event.Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to serialize event: %w", err)
	}

	res, err := r.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"payload": string(bytes),
		},
	}).Result()

	return res, err
}

func (r *RedisDriver) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	windowStart := now - window.Milliseconds()

	pipe := r.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart-1, 10))
	countCmd := pipe.ZCard(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("allow limit check failed: %w", err)
	}

	return countCmd.Val() < int64(limit), nil
}

func (r *RedisDriver) IncrAndExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	score := time.Now().UnixMilli()

	res, err := r.rdb.ZAdd(ctx, key, redis.Z{Score: float64(score), Member: strconv.FormatInt(score, 10)}).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to add entry: %w", err)
	}

	if res == 1 {
		if err := r.rdb.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, fmt.Errorf("failed to set expiry: %w", err)
		}
	}

	return r.rdb.ZCard(ctx, key).Result()
}
