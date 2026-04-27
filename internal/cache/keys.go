package cache

import (
	"fmt"
	"net"
	"strings"

	"github.com/whoiscache/whoiscache/internal/whois"
)

func DomainKey(domain string) string {
	d := strings.ToLower(strings.TrimSpace(domain))
	d = strings.TrimSuffix(d, ".")
	return "d:" + d
}

func ASNKey(asn string) string {
	return "a:" + strings.ToUpper(strings.TrimSpace(asn))
}

// IPCacheKey is the single RecordKey for an IP response (one CIDR from WHOIS, or /24 / /48 fallback).
func IPCacheKey(queryIP net.IP, whoisText []byte) string {
	is4 := queryIP.To4() != nil
	if prefix, ok := whois.BestPrefixFromWHOIS(queryIP, string(whoisText)); ok && prefix != "" {
		if k := recordKeyFromCIDR(prefix, is4); k != "" {
			return k
		}
	}
	return fallbackIPRecordKey(queryIP)
}

// IPSingleflightKey groups concurrent upstream lookups (same /24 for IPv4, /48 for IPv6).
func IPSingleflightKey(ip net.IP) string {
	return fallbackIPRecordKey(ip)
}

func fallbackIPRecordKey(ip net.IP) string {
	if ip4 := ip.To4(); ip4 != nil {
		m := ip4.Mask(net.CIDRMask(24, 32))
		return fmt.Sprintf("4:%s/24", m.String())
	}
	if len(ip) == net.IPv6len && ip.To4() == nil {
		m := ip.Mask(net.CIDRMask(48, 128))
		return fmt.Sprintf("6:%s/48", m.String())
	}
	return ""
}

func recordKeyFromCIDR(cidr string, is4 bool) string {
	cidr = strings.TrimSpace(cidr)
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil || ipnet == nil {
		return ""
	}
	if is4 {
		if ipnet.IP.To4() == nil {
			return ""
		}
		return "4:" + ipnet.String()
	}
	if ipnet.IP.To4() != nil {
		return ""
	}
	return "6:" + ipnet.String()
}
