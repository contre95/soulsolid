---
weight: 140
title: "Logging"
description: "Soulsolid's own logs: where to find them and how much detail they show."
icon: "terminal"
draft: false
toc: true
---

Soulsolid writes a log of what it's doing (starting up, requests, errors) to the
console. It's the first place to look when something doesn't work as expected.

These are the **application logs**. Each background job also writes its own log, which
you can read from the app; see [Jobs → Job logs](./jobs.md#job-logs).

## Where to find the logs

Logs are written to the standard output of the process:

- **Container**: `podman logs soulsolid` or `docker logs soulsolid`
  (add `-f` to follow them live).
- **Kubernetes**: `kubectl logs <pod>`.
- **Running it directly**: in the terminal where you started it.

Each line shows the time, the level, where it came from, and the message.

## Log levels

| Level | What you see |
|-------|--------------|
| `info` *(default)* | Normal activity, warnings and errors. Good for everyday use. |
| `debug` | Much more detail, including every request made by the web interface. Use it when investigating a problem or reporting a bug. |

## Log formats

| Format | Looks like | Use it for |
|--------|------------|------------|
| `logfmt` *(default)* | `key=value` pairs | Reading in a terminal, and most log tools. |
| `text` | Plain, human-readable text | Reading in a terminal. |
| `json` | One JSON object per line | Sending logs to a log collector (Loki, Elasticsearch, …). |

## Configuration

```yaml
logger:
  enabled: true     # turn logging on or off
  level: info       # info or debug
  format: logfmt    # logfmt, text or json
  htmx_debug: false # extra debugging information in the web interface, for troubleshooting
```

These can also be changed from the **Settings** page.

## Related

- [Jobs](./jobs.md#job-logs): per-job logs, readable from the app.
- [Configuration](./config.md): the Settings page.
