package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMinimalINI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.ini")
	content := `
[server]
listen_addr=:0
read_timeout=1s
write_timeout=1s
max_conns=1
worker_pool_size=1

[metrics]
listen_addr=:0

[storage]
snapshot_path=` + filepath.Join(dir, "snap.dat") + `
snapshot_interval=1s

[cache]
domain_ttl=1s
ip_ttl=1s
asn_ttl=1s
negative_ttl=1s
stale_ttl=1s

[whois]
default_timeout=1s
max_response_bytes=1000
domain_root_server=whois.iana.org
ip_backend_strategy=iana_referral
asn_backend_strategy=iana_referral
asn_fallback_server=whois.radb.net
max_referral_hops=2
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Whois.DomainRootServer == "" {
		t.Fatal("empty domain root")
	}
	if c.Storage.SnapshotPath == "" {
		t.Fatal("empty snapshot path")
	}
}
