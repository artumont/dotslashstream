package platform

import (
	"context"
	"time"
)

type Event interface {
	Serialize() ([]byte, error)
}

type RedisClient interface {
	// Close terminates the connection to the Redis server.
	Close() error

	// PublishEvent appends an event to the named Redis Stream and returns the event ID
	// assigned by Redis.
	PublishEvent(ctx context.Context, stream string, event Event) (string, error)

	// Allow checks if a key (e.g., IP, User ID) is within limits.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// IncrAndExpire increments a key and sets a TTL if it's a new key
	IncrAndExpire(ctx context.Context, key string, ttl time.Duration) (int64, error)
}
