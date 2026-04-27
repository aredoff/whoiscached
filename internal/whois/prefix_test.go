package whois

import (
	"net"
	"testing"
)

func TestBestPrefixFromWHOIS_InetNumRange(t *testing.T) {
	q := net.ParseIP("82.146.40.1")
	text := `inetnum:        82.146.40.0 - 82.146.47.255
netname:        EXAMPLE-RU
`
	p, ok := BestPrefixFromWHOIS(q, text)
	if !ok || p != "82.146.40.0/21" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
}

func TestBestPrefixFromWHOIS_PicksNarrowest(t *testing.T) {
	q := net.ParseIP("1.1.1.1")
	text := `
route: 1.0.0.0/8
CIDR: 1.1.0.0/16
CIDR: 1.1.1.0/24
`
	p, ok := BestPrefixFromWHOIS(q, text)
	if !ok || p != "1.1.1.0/24" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
}

func TestIsNegativeResponse(t *testing.T) {
	if !IsNegativeResponse([]byte("No match for DOMAIN")) {
		t.Fatal("expected true")
	}
}

func TestNormalizeASN(t *testing.T) {
	_, s, ok := NormalizeASN("as15169")
	if !ok || s != "AS15169" {
		t.Fatalf("got %q ok=%v", s, ok)
	}
}
