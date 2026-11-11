# Using Firefly From Another Module

This document describes the public surface a service uses when it lives in
**its own repository** and depends on `github.com/zhangpeihaoks/firefly`.

## Why a separate surface exists

Firefly keeps its engine under `internal/`:

- `internal/transport/http` — the HTTP server
- `internal/middleware` — recovery, logging, request id, timeout, tracing, metrics
- `internal/errors` — the unified error type

Go's `internal` rule means any package outside
`github.com/zhangpeihaoks/firefly/...` cannot import them. That is fine for
the examples in this repository, which are inside the module, but it makes the
framework unusable for a service in its own repository.

`pkg/server`, `pkg/middleware` and `pkg/errors` close that gap. They are thin
wrappers: no behaviour of their own, no duplicated logic, and every change to
the internals flows through automatically.

## Import surface

```go
import (
    "github.com/zhangpeihaoks/firefly/app"            // lifecycle: New / Server / Run
    "github.com/zhangpeihaoks/firefly/pkg/server"     // HTTP server
    "github.com/zhangpeihaoks/firefly/pkg/middleware" // handler + middleware types
    "github.com/zhangpeihaoks/firefly/pkg/errors"     // status-carrying errors
)
```

### pkg/server

| Symbol | Notes |
|---|---|
| `New(opts ...Option) *Server` | Does not listen until `Start` |
| `Address(string)` | Defaults to `:8080` |
| `Timeout(time.Duration)` | Read/write/shutdown timeout, defaults to 30s |
| `MaxRequestBodySize(int64)` | Rejects oversized `Content-Length` with 413 |
| `Middleware(...middleware.Middleware)` | Global chain; with none, the framework installs its default observability chain |
| `(*Server).Route(method, path, h, ms...)` | Per-route middleware runs after the global chain |
| `(*Server).Group(prefix, ms...)` | Route groups, nestable |
| `(*Server).Start(ctx) / Stop(ctx)` | Lifecycle expected by `app.Server` |
| `(*Server).Endpoint() (*url.URL, error)` | Useful in tests with `Address("127.0.0.1:0")` |

Handler signature: `func(ctx context.Context, req any) (any, error)`. Over
HTTP, `req` is the original `*http.Request`, so type-assert it when you need
the raw body, query string or headers. The return value is serialised as a
`200` JSON response.

### pkg/middleware

`Handler` and `Middleware` are type aliases of the internal types, so values
compose with `Chain`. The constructors take no options on purpose: option
types are internal and would leak into the public API.

Available: `Recovery()`, `Logging()`, `RequestID()`, `Timeout(d)`, `Chain(...)`,
and `RequestIDFromContext(ctx)`.

### pkg/errors

`New(code, reason, message)`, `Newf(code, reason, format, args...)`,
`Wrap(err, code, reason, message)`, the `*Error` type (an alias), and status
constants (`CodeBadRequest`, `CodeNotFound`, `CodeInternal`,
`CodeServiceUnavailable`, ...).

The HTTP transport turns a returned `*errors.Error` into a response with its
`Code` as the status. Any other error is masked as `500` so internal details
do not leak. The body is the unified envelope:

```json
{"code": 503, "msg": "config missing", "data": null}
```

`reason` is carried for logging and gRPC mapping, not rendered in the body.

## Limits of the public surface

- Only the HTTP transport is exposed. gRPC and WebSocket servers are still
  internal; a service that needs them must live inside this module.
- Middleware options (custom timeout paths, CORS settings, JWT/APIKey auth)
  are not re-exported yet.
- `pkg/server` does not expose the underlying router, so pprof or custom gin
  routes are not reachable from an external service.
- The internal `alert` package (Feishu/Gotify webhooks) is also internal, so an
  external service implements its own notifiers.

When a service needs more than this, prefer promoting the specific piece to
`pkg/` in a dedicated change instead of importing internals through a fork.

## Compatibility

These packages follow the module's semver tags. Before `v1.0.0` the API may
change between minor versions; pin an exact tag in `go.mod` and read the
release notes before upgrading.
