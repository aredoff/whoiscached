package cache

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Bodies under minWHOISCompressLen stay uncompressed (gzip often inflates small payloads).
const minWHOISCompressLen = 64

func isGzip(p []byte) bool {
	return len(p) >= 2 && p[0] == 0x1f && p[1] == 0x8b
}

// compressStored applies gzip for long WHOIS text; short payloads are kept as-is.
func compressStored(plain []byte) []byte {
	if len(plain) == 0 {
		return nil
	}
	if len(plain) < minWHOISCompressLen {
		return bytes.Clone(plain)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(plain); err != nil {
		_ = w.Close()
		return bytes.Clone(plain)
	}
	if err := w.Close(); err != nil {
		return bytes.Clone(plain)
	}
	out := buf.Bytes()
	if len(out) >= len(plain) {
		return bytes.Clone(plain)
	}
	return out
}

// decompressStored returns plain WHOIS from Record/snapshot; handles gzip and legacy raw.
func decompressStored(p []byte) ([]byte, error) {
	if len(p) == 0 {
		return p, nil
	}
	if !isGzip(p) {
		return p, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(p))
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		return nil, err
	}
	return out, nil
}
