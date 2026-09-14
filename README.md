# parameters-core

[![Go Reference](https://pkg.go.dev/badge/github.com/jasonmiller-cc/parameters-core.svg)](https://pkg.go.dev/github.com/jasonmiller-cc/parameters-core)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Foundational Go library shared across all **parameters** services. Provides consistent HTTP server setup, authentication, structured logging, metrics, health checks, config loading, and TLS utilities so each service can focus on its domain logic.

## Package overview

| Package | Description |
|---|---|
| `pkg/config` | YAML config loading with env var overlay (`envPrefix_VAR`) |
| `pkg/log` | `log/slog` wrapper with context propagation |
| `pkg/server` | `net/http` server factory with graceful shutdown |
| `pkg/middleware` | Request ID, structured logging, panic recovery, CORS, middleware chaining |
| `pkg/auth` | JWT signing/verification, API key middleware, role-based access |
| `pkg/response` | JSON envelope helpers (`OK`, `Created`, `Err`, `List`) |
| `pkg/errors` | Typed `*APIError` with HTTP status codes and machine-readable codes |
| `pkg/health` | Composable `/healthz` and `/livez` check framework |
| `pkg/metrics` | Prometheus registry + HTTP instrumentation middleware |
| `pkg/tlsutil` | TLS cert/key loading, mTLS config builders |
| `pkg/client` | Retry-capable HTTP client for inter-service calls |
| `pkg/version` | Build-time version embedding |
| `pkg/ui` | Service registry, health polling, and HTTP handlers backing the `dashboard` binary |

## Quick start

```go
import (
    "github.com/jasonmiller-cc/parameters-core/pkg/auth"
    "github.com/jasonmiller-cc/parameters-core/pkg/config"
    "github.com/jasonmiller-cc/parameters-core/pkg/health"
    corelog "github.com/jasonmiller-cc/parameters-core/pkg/log"
    "github.com/jasonmiller-cc/parameters-core/pkg/metrics"
    "github.com/jasonmiller-cc/parameters-core/pkg/middleware"
    "github.com/jasonmiller-cc/parameters-core/pkg/server"
)

func main() {
    log := corelog.New(corelog.LevelInfo, corelog.FormatJSON)

    var cfg config.BaseConfig
    config.Load("", "PARAMS_MYSERVICE", &cfg)

    reg := metrics.New("myservice")
    checker := health.New("myservice", version.String())
    jwtCfg := &auth.JWTConfig{Secret: []byte(cfg.Auth.JWTSecret), Issuer: cfg.Auth.JWTIssuer}

    mux := http.NewServeMux()
    mux.Handle("GET /healthz", checker.Handler())
    mux.Handle("GET /livez",   health.LiveHandler())
    mux.Handle("GET /metrics", reg.Handler())

    chain := middleware.Chain(
        middleware.RequestID,
        middleware.Logger(log),
        middleware.Recover(log),
        middleware.CORS("*"),
        reg.Middleware,
    )

    srv := server.New(cfg.Server, chain(mux), log)
    if err := srv.Run(context.Background()); err != nil {
        log.Error("server error", "err", err)
    }
}
```

## Services that use this library

- [parameters-dns](https://github.com/jasonmiller-cc/parameters-dns) — DNS zone and record management
- [parameters-dhcp](https://github.com/jasonmiller-cc/parameters-dhcp) — DHCP server management
- [parameters-ldap](https://github.com/jasonmiller-cc/parameters-ldap) — LDAP/directory management
- [parameters-kerberos](https://github.com/jasonmiller-cc/parameters-kerberos) — Kerberos KDC management
- [parameters-ca](https://github.com/jasonmiller-cc/parameters-ca) — Certificate authority and ACME
- [parameters-iam](https://github.com/jasonmiller-cc/parameters-iam) — Identity and access management
- [parameters-ntp](https://github.com/jasonmiller-cc/parameters-ntp) — NTP service management
- [parameters-fs](https://github.com/jasonmiller-cc/parameters-fs) — Filesystem and quota management
- [parameters-network](https://github.com/jasonmiller-cc/parameters-network) — Network interface and routing

## Dashboard

`cmd/dashboard` is a standalone binary that polls every registered service's
`/healthz` endpoint and serves a web UI showing platform-wide status, version,
and latency, plus a reverse proxy for calling any service's API directly from
the browser.

```bash
go build -o dashboard ./cmd/dashboard
./dashboard                # auto-discovers dashboard.yaml, else uses built-in defaults
./dashboard -config dashboard.yaml
./dashboard -gen-config > dashboard.yaml   # print a starter config
./dashboard -version
```

By default it listens on `0.0.0.0:9090` and polls `http://localhost:8081`-`8089`
(the standard per-service ports below). Config is loaded from `dashboard.yaml`
in the working directory, then `/etc/parameters/dashboard.yaml`, falling back
to `ui.DefaultConfig()` if neither exists.

```yaml
server:
  host: 0.0.0.0
  port: 9090

poll:
  interval_s: 30

services:
  - name: dns
    display_name: DNS
    url: http://localhost:8081
    description: DNS zone and record management
    group: network
```

Endpoints:

| Endpoint | Description |
|---|---|
| `GET /` | Dashboard UI |
| `GET /api/platform/summary` | Aggregate health counts |
| `GET /api/platform/services` | Per-service status, version, latency |
| `POST /api/platform/poll` | Trigger an immediate poll of all services |
| `GET /api/version` | Dashboard build version |
| `/proxy/{service}/{rest...}` | Reverse proxy to a registered service's API |

## Config conventions

Each service reads config from `/etc/parameters/<service>/config.yaml` by default.
Override with `PARAMS_<SERVICE>_CONFIG=/path/to/config.yaml`.

All `BaseConfig` fields can be overridden by environment variables using the
`env:"VAR"` tag notation with the service prefix:

```
PARAMS_DNS_SERVER_PORT=8443
PARAMS_DNS_AUTH_JWT_SECRET=...
PARAMS_DNS_LOG_LEVEL=debug
```

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).
