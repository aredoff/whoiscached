# whoiscache

A caching WHOIS server (TCP, query line + `\n`). Responses are kept **in memory** with a periodic **on-disk snapshot** (`[storage]`). Upstream follows IANA referrals; for ASN, fallback comes from the config.

## Features

- **Domain / ASN / IP**: keys `d:`, `a:`, `4:`/`6:` plus a normalized value. For IP, **one** CIDR from the WHOIS response is stored; lookup is **LPM** (most specific prefix).
- **Stale**: two TTL layers per record; on upstream failure stale is served until `stale_ttl` expires.
- **Snapshot**: written only when **dirty**; atomically (`tmp` + `rename`). Loaded from disk on startup.
- **CLI**: `-dump-keys` — keys from the snapshot; `-get-key=<RecordKey>` — primary from the snapshot; `-delete-key=<RecordKey>` — remove the record and update the snapshot (opens DiskStore, `Close` writes the file).
- **Metrics**: `GET /metrics`.

## Requirements

- Go 1.23+ (see `go.mod`)

## Run

```bash
export WHOISCACHE_CONFIG=configs/config.ini
go run ./cmd/whoiscached -config configs/config.ini
```

The directory for `snapshot_path` is created on first write.

**systemd:** [deploy/whoiscached.service](deploy/whoiscached.service) and steps in [deploy/README.md](deploy/README.md).

## Configuration

See [deploy/config.ini.example](deploy/config.ini.example).

| Section   | Purpose                                                   |
| --------- | --------------------------------------------------------- |
| `[server]`  | WHOIS TCP, timeouts, `max_conns`, `worker_pool_size`.   |
| `[metrics]` | HTTP for `/metrics`.                                   |
| `[storage]` | `snapshot_path`, `snapshot_interval`.                  |
| `[cache]`   | Domain/IP/ASN TTL, `negative_ttl`, `stale_ttl`.         |
| `[whois]`   | Upstream, limits, `iana_referral` for IP/ASN.          |

## Metrics

- `whoiscache_requests_total{kind,result}`
- `whoiscache_errors_total{stage}`
- `whoiscache_request_duration_seconds{kind}`

## Limitations

- For IP/ASN, only `iana_referral` is supported in the config.
