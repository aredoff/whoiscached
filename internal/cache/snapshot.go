package cache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var snapMagic = []byte("WHOISNP1")

const snapVersion uint32 = 2

type snapRow struct {
	Key        string
	Body       []byte
	ValidUntil time.Time
	StaleBody  []byte
	StaleUntil time.Time
}

func ReadSnapshotFile(path string) ([]snapRow, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return decodeSnapshot(b)
}

func decodeSnapshot(b []byte) ([]snapRow, error) {
	if len(b) < len(snapMagic)+4 {
		return nil, errors.New("snapshot: file too short")
	}
	if string(b[:len(snapMagic)]) != string(snapMagic) {
		return nil, errors.New("snapshot: bad magic")
	}
	off := len(snapMagic)
	ver := binary.LittleEndian.Uint32(b[off : off+4])
	off += 4
	if ver != snapVersion {
		return nil, fmt.Errorf("snapshot: unsupported version %d", ver)
	}
	if len(b) < off+8 {
		return nil, errors.New("snapshot: truncated header")
	}
	n := int(binary.LittleEndian.Uint64(b[off : off+8]))
	off += 8
	out := make([]snapRow, 0, n)
	now := time.Now()
	for range n {
		if len(b) < off+4 {
			return nil, errors.New("snapshot: truncated record")
		}
		kl := int(binary.LittleEndian.Uint32(b[off : off+4]))
		off += 4
		if len(b) < off+kl+8+4+8+4 {
			return nil, errors.New("snapshot: truncated key")
		}
		key := string(b[off : off+kl])
		off += kl
		vu := int64(binary.LittleEndian.Uint64(b[off : off+8]))
		off += 8
		vl := int(binary.LittleEndian.Uint32(b[off : off+4]))
		off += 4
		if len(b) < off+vl+8+4 {
			return nil, errors.New("snapshot: truncated body")
		}
		body := append([]byte(nil), b[off:off+vl]...)
		off += vl
		su := int64(binary.LittleEndian.Uint64(b[off : off+8]))
		off += 8
		sl := int(binary.LittleEndian.Uint32(b[off : off+4]))
		off += 4
		if len(b) < off+sl {
			return nil, errors.New("snapshot: truncated stale")
		}
		stale := append([]byte(nil), b[off:off+sl]...)
		off += sl
		bodyPlain, err := decompressStored(body)
		if err != nil {
			return nil, fmt.Errorf("snapshot: body: %w", err)
		}
		stalePlain, err := decompressStored(stale)
		if err != nil {
			return nil, fmt.Errorf("snapshot: stale: %w", err)
		}
		validUntil := time.Unix(0, vu)
		staleUntil := time.Unix(0, su)
		if now.After(validUntil) && now.After(staleUntil) {
			continue
		}
		out = append(out, snapRow{
			Key:        key,
			Body:       bodyPlain,
			ValidUntil: validUntil,
			StaleBody:  stalePlain,
			StaleUntil: staleUntil,
		})
	}
	if off != len(b) {
		return nil, errors.New("snapshot: trailing garbage")
	}
	return out, nil
}

func WriteSnapshotFile(path string, rows []snapRow) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".whoiscache-snap-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removed := false
	defer func() {
		if !removed {
			_ = os.Remove(tmpPath)
		}
	}()

	if err = encodeSnapshot(tmp, rows); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	removed = true
	return nil
}

func onDiskForm(plainOrStored []byte) []byte {
	if len(plainOrStored) == 0 {
		return nil
	}
	if isGzip(plainOrStored) {
		return plainOrStored
	}
	return compressStored(plainOrStored)
}

func encodeSnapshot(w io.Writer, rows []snapRow) error {
	if _, err := w.Write(snapMagic); err != nil {
		return err
	}
	var ver [4]byte
	binary.LittleEndian.PutUint32(ver[:], snapVersion)
	if _, err := w.Write(ver[:]); err != nil {
		return err
	}
	var count [8]byte
	binary.LittleEndian.PutUint64(count[:], uint64(len(rows)))
	if _, err := w.Write(count[:]); err != nil {
		return err
	}
	var buf [4]byte
	var ts [8]byte
	for _, row := range rows {
		body := onDiskForm(row.Body)
		st := onDiskForm(row.StaleBody)
		k := []byte(row.Key)
		binary.LittleEndian.PutUint32(buf[:], uint32(len(k)))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
		if _, err := w.Write(k); err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(ts[:], uint64(row.ValidUntil.UnixNano()))
		if _, err := w.Write(ts[:]); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[:], uint32(len(body)))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
		if _, err := w.Write(body); err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(ts[:], uint64(row.StaleUntil.UnixNano()))
		if _, err := w.Write(ts[:]); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(buf[:], uint32(len(st)))
		if _, err := w.Write(buf[:]); err != nil {
			return err
		}
		if _, err := w.Write(st); err != nil {
			return err
		}
	}
	return nil
}

func SortedSnapKeys(rows []snapRow) []string {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.Key)
	}
	sort.Strings(keys)
	return keys
}

func SnapGetPrimary(rows []snapRow, key string) ([]byte, bool) {
	now := time.Now()
	for _, r := range rows {
		if r.Key != key {
			continue
		}
		if len(r.Body) == 0 || !now.Before(r.ValidUntil) {
			return nil, false
		}
		return r.Body, true
	}
	return nil, false
}
