# whoiscache

Кеширующий WHOIS-сервер (TCP, строка запроса + `\n`). Ответы хранятся **в памяти** с периодическим **снимком на диск** (`[storage]`). Апстрим — рефералы от IANA; для ASN — fallback из конфига.

## Возможности

- **Домен / ASN / IP**: ключи `d:`, `a:`, `4:`/`6:` + нормализованное значение. Для IP после ответа сохраняется **один** CIDR из WHOIS; поиск — **LPM** (самый специфичный префикс).
- **Stale**: в каждой записи два слоя TTL; при ошибке апстрима отдаётся stale, пока не истёк `stale_ttl`.
- **Снимок**: пишется только при **dirty**; атомарно (`tmp` + `rename`). При старте загружается с диска.
- **CLI**: `-dump-keys` — ключи из снимка; `-get-key=<RecordKey>` — primary из снимка; `-delete-key=<RecordKey>` — удалить запись и обновить снимок (открывает DiskStore, `Close` пишет файл).
- **Метрики**: `GET /metrics`.

## Требования

- Go 1.23+ (см. `go.mod`)

## Запуск

```bash
export WHOISCACHE_CONFIG=configs/config.ini
go run ./cmd/whoiscached -config configs/config.ini
```

Каталог для `snapshot_path` создаётся при первой записи.

**systemd:** [deploy/whoiscached.service](deploy/whoiscached.service) и шаги в [deploy/README.md](deploy/README.md).

## Конфигурация

См. [configs/config.ini.example](configs/config.ini.example).

| Секция | Назначение |
|--------|------------|
| `[server]` | WHOIS TCP, таймауты, `max_conns`, `worker_pool_size`. |
| `[metrics]` | HTTP для `/metrics`. |
| `[storage]` | `snapshot_path`, `snapshot_interval`. |
| `[cache]` | TTL домена/IP/ASN, `negative_ttl`, `stale_ttl`. |
| `[whois]` | Апстрим, лимиты, `iana_referral` для IP/ASN. |

## Метрики

- `whoiscache_requests_total{kind,result}`
- `whoiscache_errors_total{stage}`
- `whoiscache_request_duration_seconds{kind}`

## Ограничения

- Для IP/ASN в конфиге поддержан только `iana_referral`.
