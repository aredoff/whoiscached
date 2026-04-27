package cache

import (
	"context"
	"errors"
	"net"
	"time"
)

var ErrNotFound = errors.New("cache: not found")

type Backend interface {
	Get(ctx context.Context, key string) (string, error)
	GetStale(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key, val string, ttlPrimary, ttlStale time.Duration) error
	LookupIPPrimary(ctx context.Context, ip net.IP) (string, error)
	LookupIPStale(ctx context.Context, ip net.IP) (string, error)
	Close() error
}
