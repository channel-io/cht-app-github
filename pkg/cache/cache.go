package cache

import (
	"context"
	"time"
)

type Cache[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
	Set(ctx context.Context, key string, value T, expiry time.Duration) error
}
