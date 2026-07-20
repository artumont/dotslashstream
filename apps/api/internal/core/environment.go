package core

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Environment struct {
	RedisAddr string `env:"REDIS_ADDRESS"`

	BucketAddr      string `env:"BUCKET_ADDRESS"`
	BucketKeyID     string `env:"BUCKET_KEY_ID"`
	BucketAccessKey string `env:"BUCKET_ACCESS_KEY"`

	DatabaseDSN string `env:"DB_DSN"`

	Port int `env:"PORT" envDefault:"8080"`
}

func LoadEnvironment() (*Environment, error) {
	cfg := &Environment{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse environment: %w", err)
	}

	return cfg, nil
}
