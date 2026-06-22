---
weight: 120
title: "Hosting"
description: "The Fiber web server, middleware, content negotiation, and route wiring."
icon: "dns"
draft: false
toc: true
---

The **hosting** feature is the glue that turns a pile of independent features into a
running web application. It builds the [Fiber](https://gofiber.io/) HTTP server,
registers the template engine and custom template functions, installs middleware,
wires every feature's routes in the correct order, and serves static assets. It also
owns the **`respond`** package — the content-negotiation helpers that let every
handler return either an HTMX HTML fragment or JSON from the same code path.

## What it does

- Constructs the **Fiber app** with the HTML template engine and global config
  (body limit, error handler, immutability, locals).
- Registers **custom template functions** (duration formatting, capitalize, path
  base, URL encoding, debug flag, …).
- Installs **middleware** for HTMX-aware request logging.
- Serves **static files** (`/public`, `/node_modules`) and a `/health` check.
- **Registers every feature's routes** in a deliberate order.
- Provides the **`respond`** package used by all handlers for HTMX/JSON negotiation.
- Hosts the optional **Telegram bot** wiring (`telegram.go`).

## How it works

```
server.go            → builds the Fiber app, template engine, registers routes
middleware.go        → HTMX / request-logging middleware
respond/respond.go   → content-negotiation helpers (Section, Partial, Toast*, …)
telegram.go          → Telegram bot wiring
```

### Server construction

`NewServer` (`server.go`) receives every feature's service as a parameter — this is
the single composition point where the dependency graph comes together. It:

1. Creates the `html` template engine pointed at `./views`, enabling debug rendering
   when the log level is `debug`, and registers helper functions (`duration`,
   `formatDuration`, `totalDuration`, `capitalize`, `add`, `pathBase`, `urlEncode`,
   `isDebug`).
2. Builds the Fiber app with a custom error handler, a large body limit (for uploads),
   `PassLocalsToViews`, and `Immutable` buffers.
3. Installs middleware and a locals injector that exposes `Version`, `Downloaders`,
   and `Telegram` to every template.
4. Serves `./public` and `./node_modules` statically and adds `/health`.
5. Calls each feature's `RegisterRoutes`.

`Start()` listens on the configured port; `Shutdown()` stops gracefully.

### Route registration order matters

Fiber matches routes **in registration order**, so more specific literal paths must
be registered before wildcard-parameter routes that share a prefix. The canonical
example called out in the code: lyrics registers `GET /tag/:trackId/lyrics` *before*
metadata registers `GET /tag/:trackId/:provider`, otherwise the wildcard would
swallow `/lyrics` requests. Keep this ordering in mind when adding routes.

### Middleware

`middleware.go` provides request logging:

- **`HTMXMiddleware`** — times requests, and for HTMX requests logs method/path/status
  plus the HTMX trigger/target headers. High-frequency polling endpoints
  (`/jobs/list`, `/jobs/count`, `/queue/count`) are skipped to avoid log spam.
- **`LogAllRequestsMiddleware`** — logs every request, escalating to `ERROR` level for
  status ≥ 400.
- **`HTMXDebugMiddleware`** — verbose HTMX header dumping for debugging (used when
  HTMX debug is enabled).

### The `respond` package — content negotiation

`respond` is the most reused piece of the feature; almost every handler in the app
calls one of its helpers. They branch on the `HX-Request` header (HTMX vs. plain
client) so a single endpoint serves both the web UI and JSON API clients:

| Helper | HTMX request | Non-HTMX request |
|--------|--------------|------------------|
| `Section(c, name, data)` | render `sections/<name>` fragment | render full `main` layout (sets `data["Section"]`) |
| `Partial(c, tmpl, data)` | render `tmpl` fragment | the same `data` as JSON |
| `Text(c, key, value, …)` | plain string (optional override) | `{"key","value"}` JSON |
| `ToastOk` / `ToastErr` | render toast template | `{"message"}` / `{"error"}` JSON |
| `ToastJob(c, jobID, msg)` | success toast | `202` with `{"job_id"}` JSON |
| `Resource(c, mime, url, serve)` | (negotiates on `Accept`) serve bytes | `{"type","url"}` JSON when `Accept: application/json` |
| `HTMX(c, tmpl, data)` | render `tmpl` | `406 Not Acceptable` |

`Resource` is special: it negotiates on the `Accept` header rather than `HX-Request`,
because browser resource tags (`<img>`, `<a href>`) never send `HX-Request` but still
need the binary response. This dual-response design is what the
[API reference](../api.md) documents from the client's perspective.

## Endpoints

Hosting itself registers only a few infrastructure routes; everything else belongs to
the individual features it wires up.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/health` | Liveness check | `OK` |
| GET | `/*` | Static assets from `./public` | File |
| GET | `/node_modules/*` | Vendored frontend assets | File |

## Configuration

Reads the `server` block (and a few others) from the config `Manager`:

```yaml
server:
  port: 3535        # listen port
  show_routes: false # print the route table on startup
logger:
  level: info       # "debug" enables template debug rendering
  htmx_debug: false # exposes isDebug() to templates
```

## Related

- [Configuration](./config.md) — supplies the `Manager` injected here.
- [API reference](../api.md) — the client-facing view of `respond`'s negotiation.
- Every other feature — hosting is where their routes and services are wired together.
