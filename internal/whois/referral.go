package whois

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	reWhoisServer = regexp.MustCompile(`(?i)(?:whois\s*server|referralserver|refer whois|whois:\s*|whois:\s)\s*[:=]?\s*([a-z0-9][a-z0-9.-]*[a-z0-9]|\[[0-9a-f:.]+\])`)
	reRefer       = regexp.MustCompile(`(?i)\b(?:refer|referral)\s*:\s*([a-z0-9][a-z0-9.-]*[a-z0-9])`)
	reURLWhois    = regexp.MustCompile(`(?i)whois://([^/\s]+)`)
)

func NextServer(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := reWhoisServer.FindStringSubmatch(line); len(m) > 1 {
			return HostPort(cleanHost(m[1]))
		}
		if m := reRefer.FindStringSubmatch(line); len(m) > 1 {
			return HostPort(cleanHost(m[1]))
		}
		if m := reURLWhois.FindStringSubmatch(line); len(m) > 1 {
			return HostPort(cleanHost(m[1]))
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "remarks:") && strings.Contains(lower, "whois.") {
			if u := reURLWhois.FindStringSubmatch(line); len(u) > 1 {
				return HostPort(cleanHost(u[1]))
			}
		}
	}
	s := reURLWhois.FindStringSubmatch(response)
	if len(s) > 1 {
		return HostPort(cleanHost(s[1]))
	}
	return ""
}

func cleanHost(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			return u.Hostname()
		}
	}
	if strings.HasPrefix(s, "[") {
		if h, _, err := net.SplitHostPort(s + ":43"); err == nil {
			return h
		}
	}
	return s
}
