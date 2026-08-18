# ADSBHub

Ground-station feed hub: stations post signed aircraft position reports
(ICAO, lat/lon, squawk). The service journals them and forwards to
downstream radar HTTP consumers with retry, circuit breaker, rate limit,
dead-letter, and replay.

Design notes: [PROJECT.md](PROJECT.md).

## Run

```text
set GOTOOLCHAIN=local
set CGO_ENABLED=0
go test ./...
go run ./cmd/adsbhub
```

Open http://127.0.0.1:8080/

Default station secret: `dev-station-secret`. A local radar sink is seeded
so a test report can succeed without an external URL.
