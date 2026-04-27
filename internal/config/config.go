package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

type Server struct {
	ListenAddr     string        `ini:"listen_addr"`
	ReadTimeout    time.Duration `ini:"read_timeout"`
	WriteTimeout   time.Duration `ini:"write_timeout"`
	MaxConns       int           `ini:"max_conns"`
	WorkerPoolSize int           `ini:"worker_pool_size"`
}

type Metrics struct {
	ListenAddr string `ini:"listen_addr"`
}

type Storage struct {
	SnapshotPath     string        `ini:"snapshot_path"`
	SnapshotInterval time.Duration `ini:"snapshot_interval"`
}

type Cache struct {
	DomainTTL   time.Duration `ini:"domain_ttl"`
	IPTTL       time.Duration `ini:"ip_ttl"`
	ASNTTL      time.Duration `ini:"asn_ttl"`
	NegativeTTL time.Duration `ini:"negative_ttl"`
	StaleTTL    time.Duration `ini:"stale_ttl"`
}

type Whois struct {
	DefaultTimeout     time.Duration `ini:"default_timeout"`
	MaxResponseBytes   int64         `ini:"max_response_bytes"`
	DomainRootServer   string        `ini:"domain_root_server"`
	IPBackendStrategy  string        `ini:"ip_backend_strategy"`
	ASNBackendStrategy string        `ini:"asn_backend_strategy"`
	ASNFallbackServer  string        `ini:"asn_fallback_server"`
	MaxReferralHops    int           `ini:"max_referral_hops"`
}

type Config struct {
	Server  Server  `ini:"server"`
	Metrics Metrics `ini:"metrics"`
	Storage Storage `ini:"storage"`
	Cache   Cache   `ini:"cache"`
	Whois   Whois   `ini:"whois"`
}

func Load(path string) (*Config, error) {
	f, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:         true,
		IgnoreInlineComment: false,
	}, path)
	if err != nil {
		return nil, fmt.Errorf("load ini: %w", err)
	}
	c := defaultConfig()
	if err = f.MapTo(c); err != nil {
		return nil, fmt.Errorf("map ini: %w", err)
	}
	if err = c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func LoadFromEnv(path string) (*Config, error) {
	p := strings.TrimSpace(os.Getenv("WHOISCACHE_CONFIG"))
	if p != "" {
		path = p
	}
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}
	return Load(path)
}

func defaultConfig() *Config {
	return &Config{
		Server: Server{
			ListenAddr:     ":4343",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxConns:       10000,
			WorkerPoolSize: 1024,
		},
		Metrics: Metrics{
			ListenAddr: ":8080",
		},
		Storage: Storage{
			SnapshotPath:     "data/whoiscache.snap",
			SnapshotInterval: time.Minute,
		},
		Cache: Cache{
			DomainTTL:   1 * time.Hour,
			IPTTL:       6 * time.Hour,
			ASNTTL:      12 * time.Hour,
			NegativeTTL: 1 * time.Minute,
			StaleTTL:    7 * 24 * time.Hour,
		},
		Whois: Whois{
			DefaultTimeout:     15 * time.Second,
			MaxResponseBytes:   2 << 20,
			DomainRootServer:   "whois.iana.org",
			IPBackendStrategy:  "iana_referral",
			ASNBackendStrategy: "iana_referral",
			ASNFallbackServer:  "whois.radb.net",
			MaxReferralHops:    8,
		},
	}
}

func (c *Config) validate() error {
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("server.listen_addr is required")
	}
	if c.Server.ReadTimeout <= 0 || c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server read/write timeout must be > 0")
	}
	if c.Server.MaxConns <= 0 || c.Server.WorkerPoolSize <= 0 {
		return fmt.Errorf("server max_conns and worker_pool_size must be > 0")
	}
	if strings.TrimSpace(c.Storage.SnapshotPath) == "" {
		return fmt.Errorf("storage.snapshot_path is required")
	}
	if c.Storage.SnapshotInterval <= 0 {
		return fmt.Errorf("storage.snapshot_interval must be > 0")
	}
	if c.Cache.DomainTTL <= 0 || c.Cache.IPTTL <= 0 || c.Cache.ASNTTL <= 0 {
		return fmt.Errorf("cache.*_ttl for domain, ip, asn must be > 0")
	}
	if c.Cache.NegativeTTL <= 0 {
		return fmt.Errorf("cache.negative_ttl must be > 0")
	}
	if c.Cache.StaleTTL <= 0 {
		return fmt.Errorf("cache.stale_ttl must be > 0")
	}
	if c.Whois.DefaultTimeout <= 0 {
		return fmt.Errorf("whois.default_timeout must be > 0")
	}
	if c.Whois.MaxResponseBytes <= 0 {
		return fmt.Errorf("whois.max_response_bytes must be > 0")
	}
	if c.Whois.DomainRootServer == "" {
		return fmt.Errorf("whois.domain_root_server is required")
	}
	if c.Whois.MaxReferralHops <= 0 {
		return fmt.Errorf("whois.max_referral_hops must be > 0")
	}
	return nil
}
