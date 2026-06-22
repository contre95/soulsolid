---
weight: 140
title: "Logging"
description: "The application-wide structured logger setup."
icon: "terminal"
draft: false
toc: true
---

The **logging** feature configures the application-wide structured logger used
everywhere via Go's standard `log/slog`. It is a small setup feature — a single
`SetupLogger` function — but it underpins observability across the whole app: every
`slog.Info`/`slog.Error` call in every other feature is formatted and filtered by what
this feature builds. (It is distinct from the **per-job** log files described in
[jobs](./jobs.md), which use their own dedicated loggers.)

## What it does

- Builds a global `*slog.Logger` from the `logger` config block.
- Supports three **output formats**: `logfmt` (default), `json`, and `text`.
- Supports **log levels** `info` (default) and `debug`.
- Uses the [charmbracelet/log](https://github.com/charmbracelet/log) handler for
  colorized, caller-reporting, timestamped console output (prefixed `Soulsolid`).

## How it works

```
logger.go → SetupLogger(cfg) builds and returns the *slog.Logger
```

- **`SetupLogger(cfg *config.Manager)`** reads `logger.format` and `logger.level`,
  maps them onto charmbracelet `log` options, and wraps the handler in an `slog`
  logger. It enables `ReportCaller` and `ReportTimestamp` (using `time.Kitchen`) and
  sets the prefix to `Soulsolid`. It is called once during startup; the returned
  logger is typically set as the `slog` default so package-level `slog.*` calls route
  through it.
- **`Dup(logger, msg, args...)`** is a small helper that logs a `[DUP]`-prefixed info
  message (used to flag duplicate-related events).

### Format & level mapping

| Config value | Effect |
|--------------|--------|
| `format: logfmt` | key=value console output (default / fallback) |
| `format: json` | structured JSON lines |
| `format: text` | human-readable text |
| `level: info` | info and above (default) |
| `level: debug` | includes debug logs (verbose HTMX/request logs become visible) |

> Only `info` and `debug` are wired up in `SetupLogger`; the `warn`/`error`/`fatal`
> branches are present but commented out in the source.

## Endpoints

The logging feature registers **no HTTP routes** — it is pure setup. Log *output* is
to stderr; per-job logs are exposed through the [jobs](./jobs.md) feature's
`/jobs/:id/logs` endpoint.

## Configuration

Reads the `logger` block from the global config:

```yaml
logger:
  enabled: true     # toggle logging
  level: info       # "info" or "debug"
  format: logfmt    # "logfmt" (default), "json", or "text"
  htmx_debug: false # surfaces extra HTMX debug info (used by hosting middleware/templates)
```

The `level: debug` setting also enables verbose request/HTMX logging in the
[hosting](./hosting.md) middleware and template debug rendering.

## Related

- [Hosting](./hosting.md) — request/HTMX logging middleware that emits through this logger.
- [Jobs](./jobs.md) — separate per-job log files and the log-viewing endpoint.
- [Configuration](./config.md) — the `logger` config block.
