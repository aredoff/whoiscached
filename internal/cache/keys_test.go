package cache

import (
	"net"
	"testing"
)

func TestDomainKey(t *testing.T) {
	if got := DomainKey("EXAMPLE.com."); got != "d:example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestIPCacheKey_InetNum(t *testing.T) {
	ip := net.ParseIP("82.146.40.1")
	text := []byte(`inetnum:        82.146.40.0 - 82.146.47.255
`)
	if got := IPCacheKey(ip, text); got != "4:82.146.40.0/21" {
		t.Fatalf("got %q", got)
	}
}

func TestIPSingleflightKey(t *testing.T) {
	ip := net.ParseIP("1.2.3.4")
	if got := IPSingleflightKey(ip); got != "4:1.2.3.0/24" {
		t.Fatalf("got %q", got)
	}
}

func TestASNKey(t *testing.T) {
	if got := ASNKey("AS123"); got != "a:AS123" {
		t.Fatalf("got %q", got)
	}
}
