package whois

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/whoiscache/whoiscache/internal/config"
)

type Resolver struct {
	Client   *Client
	RootIANA string
	MaxHops  int
	Timeout  time.Duration
	Conf     *config.Whois
}

func (r *Resolver) ResolveDomain(ctx context.Context, domain string) ([]byte, error) {
	root := HostPort(r.RootIANA)
	if root == "" {
		root = "whois.iana.org:43"
	}
	return r.follow(ctx, root, domain)
}

func (r *Resolver) ResolveIP(ctx context.Context, ip string) ([]byte, error) {
	if !strings.EqualFold(strings.TrimSpace(r.Conf.IPBackendStrategy), "iana_referral") {
		return nil, fmt.Errorf("unsupported ip_backend_strategy %q", r.Conf.IPBackendStrategy)
	}
	root := HostPort(r.RootIANA)
	if root == "" {
		root = "whois.iana.org:43"
	}
	return r.follow(ctx, root, ip)
}

func (r *Resolver) ResolveASN(ctx context.Context, asn string) ([]byte, error) {
	if !strings.EqualFold(strings.TrimSpace(r.Conf.ASNBackendStrategy), "iana_referral") {
		return nil, fmt.Errorf("unsupported asn_backend_strategy %q", r.Conf.ASNBackendStrategy)
	}
	root := HostPort(r.RootIANA)
	if root == "" {
		root = "whois.iana.org:43"
	}
	q := strings.TrimSpace(asn)
	if !strings.HasPrefix(strings.ToUpper(q), "AS") {
		q = "AS" + q
	}
	b, err := r.follow(ctx, root, q)
	needFB := err != nil || IsNegativeResponse(b)
	if !needFB {
		return b, nil
	}
	fb := HostPort(r.Conf.ASNFallbackServer)
	if fb == "" {
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, fmt.Errorf("empty asn whois")
		}
		return b, nil
	}
	b2, err2 := r.follow(ctx, fb, q)
	if err2 == nil {
		return b2, nil
	}
	if err != nil {
		return b, err
	}
	if b == nil {
		return nil, err2
	}
	return b, nil
}

func (r *Resolver) follow(ctx context.Context, startServer, query string) ([]byte, error) {
	cur := startServer
	seen := map[string]struct{}{}
	for hop := 0; hop < r.MaxHops; hop++ {
		if cur == "" {
			return nil, fmt.Errorf("empty whois server")
		}
		sk := strings.ToLower(cur)
		if _, ok := seen[sk]; ok {
			break
		}
		seen[sk] = struct{}{}

		resp, err := r.Client.Query(ctx, cur, query, r.Timeout)
		if err != nil {
			return nil, err
		}
		next := NextServer(string(resp))
		if next == "" || sameWhoisTarget(cur, next) {
			return resp, nil
		}
		cur = next
	}
	return nil, fmt.Errorf("whois referral loop or max hops")
}

func sameWhoisTarget(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
