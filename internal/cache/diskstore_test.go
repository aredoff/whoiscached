package cache

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskStore_LPM_PrefersMoreSpecific(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.snap")
	s, err := NewDiskStore(path, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	_ = s.Put(ctx, "4:10.0.0.0/16", "sixteen", time.Hour, time.Hour)
	_ = s.Put(ctx, "4:10.0.1.0/24", "twentyfour", time.Hour, time.Hour)
	ip := net.ParseIP("10.0.1.5")
	v, err := s.LookupIPPrimary(ctx, ip)
	if err != nil {
		t.Fatal(err)
	}
	if v != "twentyfour" {
		t.Fatalf("got %q want twentyfour", v)
	}
}

func TestDiskStore_PruneAndDirtySnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s2.snap")
	s, err := NewDiskStore(path, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	if err := s.Put(ctx, "d:example.test", "body", 20*time.Millisecond, 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	_, _ = s.Get(ctx, "d:example.test")
	if err := s.snapshotWriteIfDirty(); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadSnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Key == "d:example.test" {
			t.Fatalf("expected key pruned from snapshot, still present")
		}
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rt.snap")
	s, err := NewDiskStore(path, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_ = s.Put(ctx, "a:AS1", "asn", time.Hour, time.Hour)
	if err := s.snapshotWriteIfDirty(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := NewDiskStore(path, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s2.Close() }()
	v, err := s2.Get(ctx, "a:AS1")
	if err != nil || v != "asn" {
		t.Fatalf("got %q %v", v, err)
	}
}

func TestDiskStore_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.snap")
	s, err := NewDiskStore(path, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "d:del.test", "x", time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "d:del.test"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "d:del.test"); err == nil {
		t.Fatal("expected ErrNotFound")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadSnapshotFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Key == "d:del.test" {
			t.Fatal("key still in snapshot")
		}
	}
}

func TestReadSnapshotMissing(t *testing.T) {
	rows, err := ReadSnapshotFile(filepath.Join(t.TempDir(), "nope.snap"))
	if err != nil || rows != nil {
		t.Fatalf("got %v %#v", err, rows)
	}
}

func TestEncodeDecodeSnapshot(t *testing.T) {
	now := time.Now().Add(time.Hour)
	rows := []snapRow{
		{Key: "d:x", Body: []byte("b"), ValidUntil: now, StaleBody: []byte("b"), StaleUntil: now},
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "e.snap")
	if err := WriteSnapshotFile(p, rows); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out, err := decodeSnapshot(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Key != "d:x" {
		t.Fatalf("%#v", out)
	}
}
