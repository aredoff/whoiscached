package service

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/whoiscache/whoiscache/internal/cache"
	"github.com/whoiscache/whoiscache/internal/config"
	"github.com/whoiscache/whoiscache/internal/whois"
)

type stubBackend struct{}

func (stubBackend) Get(context.Context, string) (string, error)      { return "", cache.ErrNotFound }
func (stubBackend) GetStale(context.Context, string) (string, error) { return "", cache.ErrNotFound }
func (stubBackend) Put(context.Context, string, string, time.Duration, time.Duration) error {
	return nil
}
func (stubBackend) LookupIPPrimary(context.Context, net.IP) (string, error) {
	return "", cache.ErrNotFound
}
func (stubBackend) LookupIPStale(context.Context, net.IP) (string, error) {
	return "", cache.ErrNotFound
}
func (stubBackend) Close() error { return nil }

func TestQueryKind(t *testing.T) {
	cfg := &config.Config{}
	r := &whois.Resolver{Conf: &cfg.Whois}
	s := New(cfg, r, stubBackend{})

	if s.QueryKind("1.1.1.1") != "ip" {
		t.Fatal("ip")
	}
	if s.QueryKind("AS15169") != "asn" {
		t.Fatal("asn")
	}
	if s.QueryKind("15169") != "asn" {
		t.Fatal("asn short")
	}
	if s.QueryKind("example.com") != "domain" {
		t.Fatal("domain")
	}
}
