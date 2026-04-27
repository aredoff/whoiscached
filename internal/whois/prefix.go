package whois

import (
	"encoding/binary"
	"math/bits"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	reCIDR    = regexp.MustCompile(`\b(?P<cidr>(?:\d{1,3}\.){3}\d{1,3}/\d{1,2})\b`)
	reCIDR6   = regexp.MustCompile(`\b(?P<cidr>[0-9a-fA-F:]{2,}:[0-9a-fA-F:%]*/\d{1,3})\b`)
	reRoute   = regexp.MustCompile(`(?i)(?:route|route6)\s*:\s*([0-9a-fA-F:./\s-]+)`)
	reInetNum = regexp.MustCompile(`(?i)inetnum\s*:\s*(\d{1,3}(?:\.\d{1,3}){3})\s*-\s*(\d{1,3}(?:\.\d{1,3}){3})\s*$`)
)

func BestPrefixFromWHOIS(queryIP net.IP, text string) (string, bool) {
	qip := queryIP
	is4 := qip.To4() != nil
	var best *net.IPNet
	var bestOnes int

	tryCIDR := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		_, ipnet, err := net.ParseCIDR(s)
		if err != nil || ipnet == nil {
			return
		}
		if is4 && ipnet.IP.To4() == nil {
			return
		}
		if !is4 && ipnet.IP.To4() != nil {
			return
		}
		if !ipnet.Contains(qip) {
			return
		}
		ones, _ := ipnet.Mask.Size()
		if best == nil || ones > bestOnes {
			best = ipnet
			bestOnes = ones
		}
	}

	for _, m := range reCIDR.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			tryCIDR(m[1])
		}
	}
	for _, m := range reCIDR6.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			tryCIDR(m[1])
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if m := reRoute.FindStringSubmatch(line); len(m) > 1 {
			p := strings.TrimSpace(m[1])
			if strings.Contains(p, "/") {
				tryCIDR(p)
			}
		}
	}
	if is4 {
		for _, line := range strings.Split(text, "\n") {
			if m := reInetNum.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 3 {
				s := net.ParseIP(m[1])
				e := net.ParseIP(m[2])
				if s == nil || e == nil {
					continue
				}
				cidr, ok := cidrFromInclusiveIPv4Range(s, e)
				if !ok {
					continue
				}
				tryCIDR(cidr)
			}
		}
	}

	if best != nil {
		return best.String(), true
	}
	return "", false
}

func cidrFromInclusiveIPv4Range(start, end net.IP) (string, bool) {
	sa := start.To4()
	ea := end.To4()
	if sa == nil || ea == nil {
		return "", false
	}
	a := binary.BigEndian.Uint32(sa)
	b := binary.BigEndian.Uint32(ea)
	if a > b {
		a, b = b, a
	}
	n := uint64(b) - uint64(a) + 1
	if n == 0 {
		return "", false
	}
	if n == 1<<32 {
		if a != 0 {
			return "", false
		}
		return "0.0.0.0/0", true
	}
	nu := uint32(n)
	if uint64(nu) != n {
		return "", false
	}
	if nu&(nu-1) != 0 {
		return "", false
	}
	if a&^(nu-1) != a {
		return "", false
	}
	pfx := 32 - bits.TrailingZeros32(nu)
	m := net.CIDRMask(pfx, 32)
	ipnet := net.IPNet{IP: sa.Mask(m), Mask: m}
	return ipnet.String(), true
}

func IsNegativeResponse(b []byte) bool {
	lower := strings.ToLower(string(b))
	neg := []string{
		"no match",
		"not found",
		"not allocated",
		"no whois server",
		"no data found",
		"0 objects found",
	}
	for _, n := range neg {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func ParseIPQuery(s string) (net.IP, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, false
	}
	return ip, true
}

func NormalizeASN(s string) (int, string, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	s = strings.TrimPrefix(s, "AS")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, "", false
	}
	return n, "AS" + strconv.Itoa(n), true
}
