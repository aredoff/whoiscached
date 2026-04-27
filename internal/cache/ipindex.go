package cache

import (
	"net"
	"sort"
	"strings"
)

type ipRoute struct {
	key  string
	net  *net.IPNet
	bits int
}

type ipIndex struct {
	v4 []ipRoute
	v6 []ipRoute
}

func (idx *ipIndex) rebuildFromKeys(keys []string, getRecord func(key string) *Record) {
	idx.v4 = idx.v4[:0]
	idx.v6 = idx.v6[:0]
	for _, k := range keys {
		if getRecord(k) == nil {
			continue
		}
		ipn, bits, is4, ok := parseRecordIPCIDR(k)
		if !ok {
			continue
		}
		r := ipRoute{key: k, net: ipn, bits: bits}
		if is4 {
			idx.v4 = append(idx.v4, r)
		} else {
			idx.v6 = append(idx.v6, r)
		}
	}
	sort.Slice(idx.v4, func(i, j int) bool { return idx.v4[i].bits > idx.v4[j].bits })
	sort.Slice(idx.v6, func(i, j int) bool { return idx.v6[i].bits > idx.v6[j].bits })
}

func parseRecordIPCIDR(key string) (ipnet *net.IPNet, bits int, is4 bool, ok bool) {
	switch {
	case strings.HasPrefix(key, "4:"):
		_, n, err := net.ParseCIDR(key[2:])
		if err != nil || n == nil || n.IP.To4() == nil {
			return nil, 0, false, false
		}
		ones, _ := n.Mask.Size()
		return n, ones, true, true
	case strings.HasPrefix(key, "6:"):
		_, n, err := net.ParseCIDR(key[2:])
		if err != nil || n == nil || n.IP.To4() != nil {
			return nil, 0, false, false
		}
		ones, _ := n.Mask.Size()
		return n, ones, false, true
	default:
		return nil, 0, false, false
	}
}

func (idx *ipIndex) lookupKey(ip net.IP) string {
	if ip4 := ip.To4(); ip4 != nil {
		for _, r := range idx.v4 {
			if r.net != nil && r.net.Contains(ip) {
				return r.key
			}
		}
		return ""
	}
	if len(ip) == net.IPv6len && ip.To4() == nil {
		for _, r := range idx.v6 {
			if r.net != nil && r.net.Contains(ip) {
				return r.key
			}
		}
	}
	return ""
}
