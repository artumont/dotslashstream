package app

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"prod"`

	RedisAddr string `env:"REDIS_ADDRESS,required"`

	BucketAddr      string `env:"BUCKET_ADDRESS,required"`
	BucketKeyID     string `env:"BUCKET_KEY_ID,required"`
	BucketAccessKey string `env:"BUCKET_ACCESS_KEY,required"`

	DatabaseDSN string `env:"DB_DSN,required"`

	Port   int  `env:"PORT" envDefault:"8080"`
	UseSSL bool `env:"USE_SSL" envDefault:"false"`

	HmacSecret string `env:"HMAC_SECRET,required,notEmpty"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse environment: %w", err)
	}

	return cfg, nil
}
