package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/logger"
)

const pingTimeout = 3 * time.Second

type Repository struct {
	client redis.UniversalClient
}

func NewRepository(client redis.UniversalClient) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Enabled() bool {
	return r != nil && r.client != nil
}

func (r *Repository) Client() redis.UniversalClient {
	if r == nil {
		return nil
	}
	return r.client
}

func NewClient(ctx context.Context, cfg *config.Config, log logger.Logger) (redis.UniversalClient, error) {
	if cfg.Valkey.URL == "" {
		log.Infow("ValKey client disabled", "logger", "valkey")
		return nil, nil
	}

	client := redis.NewUniversalClient(NewUniversalOptions(cfg))
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping valkey: %w", err)
	}

	log.Infow(
		"ValKey client enabled",
		"logger", "valkey",
		"cluster_mode", cfg.Valkey.ClusterMode,
	)

	return client, nil
}

func NewUniversalOptions(cfg *config.Config) *redis.UniversalOptions {
	return &redis.UniversalOptions{
		Addrs:         []string{cfg.Valkey.URL},
		IsClusterMode: cfg.Valkey.ClusterMode,
	}
}
