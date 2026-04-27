package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/whoiscache/whoiscache/internal/cache"
	"github.com/whoiscache/whoiscache/internal/config"
	"github.com/whoiscache/whoiscache/internal/whois"
)

var reASQuery = regexp.MustCompile(`(?i)^(AS)?[0-9]+$`)

type QueryService struct {
	cfg *config.Config
	res *whois.Resolver
	sf  singleflight.Group
	cch cache.Backend
}

func New(cfg *config.Config, r *whois.Resolver, c cache.Backend) *QueryService {
	return &QueryService{cfg: cfg, res: r, cch: c}
}

type kind int

const (
	kindDomain kind = iota
	kindIP
	kindASN
)

func (s *QueryService) QueryKind(raw string) string {
	q := cleanQuery(raw)
	if q == "" {
		return "invalid"
	}
	k, _, err := s.classify(q)
	if err != nil {
		return "invalid"
	}
	switch k {
	case kindDomain:
		return "domain"
	case kindIP:
		return "ip"
	case kindASN:
		return "asn"
	default:
		return "invalid"
	}
}

func (s *QueryService) Lookup(ctx context.Context, raw string) ([]byte, error) {
	q := cleanQuery(raw)
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}
	k, p, err := s.classify(q)
	if err != nil {
		return nil, err
	}

	switch k {
	case kindDomain:
		return s.lookupDomain(ctx, p)
	case kindIP:
		return s.lookupIP(ctx, p)
	case kindASN:
		return s.lookupASN(ctx, p)
	default:
		return nil, fmt.Errorf("unknown kind")
	}
}

func (s *QueryService) lookupDomain(ctx context.Context, d string) ([]byte, error) {
	key := cache.DomainKey(d)
	if b, ok := s.getCached(ctx, key); ok {
		return b, nil
	}
	out, err, _ := s.sf.Do(key, func() (any, error) {
		if b, ok := s.getCached(ctx, key); ok {
			return b, nil
		}
		resp, e := s.res.ResolveDomain(ctx, d)
		if e != nil {
			if st := s.getStale(ctx, key); st != nil {
				return st, nil
			}
			return nil, e
		}
		ttl := s.cfg.Cache.DomainTTL
		if whois.IsNegativeResponse(resp) {
			ttl = s.cfg.Cache.NegativeTTL
		}
		if e := s.put(ctx, key, string(resp), ttl); e != nil {
			return nil, e
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return out.([]byte), nil
}

func (s *QueryService) lookupIP(ctx context.Context, ipstr string) ([]byte, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip")
	}
	primary := cache.IPSingleflightKey(ip)
	if primary == "" {
		return nil, fmt.Errorf("no cache keys for ip")
	}
	if b, ok := s.getIPCached(ctx, ip); ok {
		return b, nil
	}
	out, err, _ := s.sf.Do(primary, func() (any, error) {
		if b, ok := s.getIPCached(ctx, ip); ok {
			return b, nil
		}
		resp, e := s.res.ResolveIP(ctx, ip.String())
		if e != nil {
			if st, se := s.cch.LookupIPStale(ctx, ip); se == nil {
				return []byte(st), nil
			}
			return nil, e
		}
		ttl := s.cfg.Cache.IPTTL
		if whois.IsNegativeResponse(resp) {
			ttl = s.cfg.Cache.NegativeTTL
		}
		key := cache.IPCacheKey(ip, resp)
		if key == "" {
			key = primary
		}
		if e := s.put(ctx, key, string(resp), ttl); e != nil {
			return nil, e
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return out.([]byte), nil
}

func (s *QueryService) lookupASN(ctx context.Context, asnRaw string) ([]byte, error) {
	_, canon, ok := whois.NormalizeASN(asnRaw)
	if !ok {
		return nil, fmt.Errorf("invalid asn")
	}
	key := cache.ASNKey(canon)
	if b, ok := s.getCached(ctx, key); ok {
		return b, nil
	}
	out, err, _ := s.sf.Do(key, func() (any, error) {
		if b, ok := s.getCached(ctx, key); ok {
			return b, nil
		}
		resp, e := s.res.ResolveASN(ctx, canon)
		if e != nil {
			if st := s.getStale(ctx, key); st != nil {
				return st, nil
			}
			return nil, e
		}
		ttl := s.cfg.Cache.ASNTTL
		if whois.IsNegativeResponse(resp) {
			ttl = s.cfg.Cache.NegativeTTL
		}
		if e := s.put(ctx, key, string(resp), ttl); e != nil {
			return nil, e
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return out.([]byte), nil
}

func (s *QueryService) getCached(ctx context.Context, key string) ([]byte, bool) {
	v, err := s.cch.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, cache.ErrNotFound) {
			_ = err
		}
		return nil, false
	}
	return []byte(v), true
}

func (s *QueryService) getIPCached(ctx context.Context, ip net.IP) ([]byte, bool) {
	v, err := s.cch.LookupIPPrimary(ctx, ip)
	if err != nil {
		if !errors.Is(err, cache.ErrNotFound) {
			_ = err
		}
		return nil, false
	}
	return []byte(v), true
}

func (s *QueryService) getStale(ctx context.Context, key string) []byte {
	v, err := s.cch.GetStale(ctx, key)
	if err != nil {
		return nil
	}
	return []byte(v)
}

func (s *QueryService) put(ctx context.Context, key, val string, ttlPrimary time.Duration) error {
	return s.cch.Put(ctx, key, val, ttlPrimary, s.cfg.Cache.StaleTTL)
}

func (s *QueryService) classify(q string) (kind, string, error) {
	if ip, ok := whois.ParseIPQuery(q); ok {
		return kindIP, ip.String(), nil
	}
	if reASQuery.MatchString(q) {
		return kindASN, q, nil
	}
	if isLikelyDomain(q) {
		return kindDomain, strings.TrimSpace(q), nil
	}
	return 0, "", fmt.Errorf("unrecognized query")
}

func isLikelyDomain(s string) bool {
	if len(s) > 253 || len(s) < 1 {
		return false
	}
	if strings.Contains(s, " ") {
		return false
	}
	if net.ParseIP(s) != nil {
		return false
	}
	if strings.Count(s, ":") > 1 {
		return false
	}
	return true
}

func cleanQuery(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
