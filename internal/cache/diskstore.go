package cache

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

type DiskStore struct {
	mu        sync.RWMutex
	records   map[string]*Record
	ipIdx     ipIndex
	path      string
	interval  time.Duration
	dirty     bool
	stop      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
}

func NewDiskStore(snapshotPath string, snapshotInterval time.Duration) (*DiskStore, error) {
	s := &DiskStore{
		records:  make(map[string]*Record),
		path:     snapshotPath,
		interval: snapshotInterval,
		stop:     make(chan struct{}),
		closed:   make(chan struct{}),
	}
	rows, err := ReadSnapshotFile(snapshotPath)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for _, row := range rows {
		if now.After(row.ValidUntil) && now.After(row.StaleUntil) {
			continue
		}
		s.records[row.Key] = &Record{
			Body:       compressStored(row.Body),
			ValidUntil: row.ValidUntil,
			StaleBody:  compressStored(row.StaleBody),
			StaleUntil: row.StaleUntil,
		}
	}
	s.rebuildIPIndexLocked()
	s.wg.Add(1)
	go s.loop()
	return s, nil
}

func (s *DiskStore) rebuildIPIndexLocked() {
	keys := make([]string, 0, len(s.records))
	for k := range s.records {
		keys = append(keys, k)
	}
	s.ipIdx.rebuildFromKeys(keys, func(key string) *Record { return s.records[key] })
}

func (s *DiskStore) pruneExpiredLocked(now time.Time) {
	changed := false
	for k, rec := range s.records {
		if rec == nil {
			delete(s.records, k)
			changed = true
			continue
		}
		if now.After(rec.ValidUntil) && now.After(rec.StaleUntil) {
			delete(s.records, k)
			changed = true
		}
	}
	if changed {
		s.rebuildIPIndexLocked()
		s.dirty = true
	}
}

func (s *DiskStore) Get(ctx context.Context, key string) (string, error) {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
	rec := s.records[key]
	if rec == nil || !rec.primaryOK(now) {
		return "", ErrNotFound
	}
	out, err := decompressStored(rec.Body)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *DiskStore) GetStale(ctx context.Context, key string) (string, error) {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
	rec := s.records[key]
	if rec == nil || !rec.staleOK(now) {
		return "", ErrNotFound
	}
	out, err := decompressStored(rec.StaleBody)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *DiskStore) LookupIPPrimary(ctx context.Context, ip net.IP) (string, error) {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
	k := s.ipIdx.lookupKey(ip)
	if k == "" {
		return "", ErrNotFound
	}
	rec := s.records[k]
	if rec == nil || !rec.primaryOK(now) {
		return "", ErrNotFound
	}
	out, err := decompressStored(rec.Body)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *DiskStore) LookupIPStale(ctx context.Context, ip net.IP) (string, error) {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
	k := s.ipIdx.lookupKey(ip)
	if k == "" {
		return "", ErrNotFound
	}
	rec := s.records[k]
	if rec == nil || !rec.staleOK(now) {
		return "", ErrNotFound
	}
	out, err := decompressStored(rec.StaleBody)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *DiskStore) Put(ctx context.Context, key, val string, ttlPrimary, ttlStale time.Duration) error {
	_ = ctx
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	p := []byte(val)
	rec := &Record{
		Body:       compressStored(p),
		ValidUntil: now.Add(ttlPrimary),
		StaleBody:  compressStored(p),
		StaleUntil: now.Add(ttlStale),
	}
	s.records[key] = rec
	if strings.HasPrefix(key, "4:") || strings.HasPrefix(key, "6:") {
		s.rebuildIPIndexLocked()
	}
	s.dirty = true
	return nil
}

// Delete removes a RecordKey from memory and marks the store dirty. Snapshot is updated on Close (or next periodic flush).
func (s *DiskStore) Delete(ctx context.Context, key string) error {
	_ = ctx
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[key]; !ok {
		return ErrNotFound
	}
	delete(s.records, key)
	if strings.HasPrefix(key, "4:") || strings.HasPrefix(key, "6:") {
		s.rebuildIPIndexLocked()
	}
	s.dirty = true
	return nil
}

func (s *DiskStore) loop() {
	defer s.wg.Done()
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			_ = s.snapshotWriteIfDirty()
		}
	}
}

func (s *DiskStore) snapshotWriteIfDirty() error {
	s.mu.Lock()
	now := time.Now()
	s.pruneExpiredLocked(now)
	if !s.dirty {
		s.mu.Unlock()
		return nil
	}
	rows := make([]snapRow, 0, len(s.records))
	for k, rec := range s.records {
		if rec == nil {
			continue
		}
		rows = append(rows, snapRow{
			Key:        k,
			Body:       rec.Body,
			ValidUntil: rec.ValidUntil,
			StaleBody:  rec.StaleBody,
			StaleUntil: rec.StaleUntil,
		})
	}
	s.mu.Unlock()

	if err := WriteSnapshotFile(s.path, rows); err != nil {
		return err
	}
	s.mu.Lock()
	s.dirty = false
	s.mu.Unlock()
	return nil
}

func (s *DiskStore) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.stop)
		s.wg.Wait()
		err = s.snapshotWriteIfDirty()
		close(s.closed)
	})
	return err
}
