package whois

import "testing"

func TestNextServer(t *testing.T) {
	next := NextServer("Whois Server: whois.apnic.net\n")
	if next == "" {
		t.Fatal("empty")
	}
	if want := "whois.apnic.net:43"; next != want {
		t.Fatalf("got %q want %q", next, want)
	}
}
