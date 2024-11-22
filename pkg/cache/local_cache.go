package cache

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

func NewLocalCache[T any]() LocalCache[T] {
	return LocalCache[T]{
		cache.New(10*time.Minute, 20*time.Minute),
	}
}

type LocalCache[T any] struct {
	*cache.Cache
}

func (c LocalCache[T]) Get(_ context.Context, key string) (*T, error) {
	res, hit := c.Cache.Get(key)
	if !hit {
		return nil, nil
	}

	t, ok := res.(T)
	if !ok {
		return nil, errors.New("Invalid value type")
	}

	return &t, nil
}

func (c LocalCache[T]) Set(_ context.Context, key string, value T, expiry time.Duration) error {
	c.Cache.Set(key, value, expiry)
	return nil
}
