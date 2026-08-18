# ADSBHub

ADS-B ground-station feed hub: accept HMAC-signed aircraft reports from
feeders, fan them out to registered radar HTTP consumers, and keep a
journal so operators can replay failed forwards.

This repository is a **runnable healthy project**. It does not ship
pre-planted defects on `main`.

## 1. Why this product

This is **aviation telemetry forwarding infrastructure**, not:

- a game or graphics demo
- a travel diary or itinerary
- a movie catalog
- CRUD-for-CRUD: shop, RBAC, inventory, OA, clinic, food delivery, CRM, tickets

Product boundary: a report arrives from a ground station, is authenticated,
routed by kind (for example `adsb.position`), then **this process POSTs it
to someone else's radar HTTP API**. The operator console only manages
radars, journal, DLQ, and replay.

## 2. Roles and happy path

| Role | Action |
|------|--------|
| Ground station | POST `/api/v1/reports` with HMAC headers, ICAO, lat/lon, squawk |
| Radar consumer | HTTP endpoint registered by the operator; receives signed JSON |
| Operator | Console at `/` to register radars, inspect journal, replay DLQ |

1. Operator registers a radar (URL + outbound secret + kind prefixes).
2. Station posts a signed report (`kind`, `icao`, `lat`, `lon`, `squawk`).
3. Hub verifies signature window, nonce, and idempotency, then fans out.
4. Worker POSTs via the feeder client; success is journaled; retryable
   failures back off; terminal failures enter DLQ.
5. Operator replays a failed forward from the journal or DLQ.

## 3. Rules that are actually implemented

### 3.1 Inbound signature

- HMAC-SHA256 over canonical `ads1.{unix}.{nonce}.{sha256_hex(body)}`.
- Header scheme `ads1=<hex>`.
- Headers: `X-Adsb-Timestamp`, `X-Adsb-Nonce`, `X-Adsb-Signature`.
- Station id: `X-Adsb-Station-Key` (default `station`).
- Window ±300s. Same nonce cannot succeed twice inside the window.

### 3.2 Idempotency

- Required `Idempotency-Key` (8–128 chars).
- Same key + same body hash: return the first accept, no second enqueue.
- Same key + different hash: 409 Conflict.

### 3.3 Radar fan-out

- Radar: id, name, URL, secret, enabled, kind prefixes, ordered, token bucket.
- Kind `adsb.position` matches prefix `adsb` or `adsb.` (segment boundary).
- One inbound report may fan out to many radars.

### 3.4 Feeder outbound POST

- JSON body, signed with the **radar** secret.
- Extra headers: `X-Adsb-Report-Id`, `X-Adsb-Forward-Id`, `X-Adsb-Attempt`, `X-Adsb-Radar`.
- 2xx success. Retryable: 408, 429, 5xx, timeout, selected net errors.
- Terminal: other 4xx including 422.

### 3.5 Retry / circuit / rate limit / DLQ

- Full-jitter exponential backoff.
- Per-radar breaker (closed / open / half-open).
- Per-radar token bucket; empty bucket delays without consuming an attempt.
- Terminal or exhausted attempts → DLQ. Replay builds a new forward id.

### 3.6 Beast-like parse

JSON reports are the primary ingest. A Mode-S Beast hex frame
(`*...;`) is also accepted and decoded into ICAO / altitude / squawk
when the bits are present.

## 4. HTTP

Control plane (unsigned, local demo):

- `GET /api/v1/healthz`
- `GET /api/v1/meta`
- `GET|POST /api/v1/radars`
- `POST /api/v1/radars/{id}/enable`
- `GET /api/v1/journal`
- `GET /api/v1/dlq`
- `POST /api/v1/replay/{forward_id}`
- `GET /api/v1/circuits`

Data plane:

- `POST /api/v1/reports`

Built-in sink: `POST /api/v1/sink` (seeded radar points here).

## 5. Run

```text
set GOTOOLCHAIN=local
set CGO_ENABLED=0
go test ./...
go run ./cmd/adsbhub
```

| Variable | Default |
|----------|---------|
| `ADSBHUB_ADDR` | `:8080` |
| `ADSBHUB_DATA_DIR` | `./data` |
| `ADSBHUB_STATION_SECRET` | `dev-station-secret` |
| `ADSBHUB_WINDOW_SEC` | `300` |
