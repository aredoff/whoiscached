package cache

import (
	"strings"
	"testing"
)

func TestCompressStoredLongText(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("Domain: example.com\n", 100)
	c := compressStored([]byte(long))
	if len(c) >= len(long) {
		t.Fatal("expected gzip to shrink this payload")
	}
	p, err := decompressStored(c)
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != long {
		t.Fatalf("round-trip mismatch")
	}
}

func TestCompressStoredShort(t *testing.T) {
	t.Parallel()
	p := []byte("x")
	if got := compressStored(p); string(got) != "x" {
		t.Fatalf("short payload stays raw")
	}
}
