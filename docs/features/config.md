---
weight: 110
title: "Configuration"
description: "Loading, validating, editing, and persisting the application config."
icon: "settings"
draft: false
toc: true
---

The **config** feature owns the application's settings. It loads the YAML config on
startup (resolving environment-variable references), validates it, exposes a
thread-safe `Manager` that every other feature reads from, renders the Settings page,
and persists edits made through the UI back to disk. It is effectively the central
nervous system for runtime configuration: paths, import behaviour, providers,
logging, jobs, and the Telegram bot all read their settings here.

## What it does

- **Loads** `config.yaml` on startup, expands `!env_var` references, and **validates**
  required fields.
- Creates a **default config** the first time the app runs if no file exists.
- Provides a thread-safe **`Manager`** (`Get`, `Update`, `Save`) injected into nearly
  every other feature.
- Renders the **Settings** section and a config form, and applies form submissions.
- Exposes the config as **JSON** or **raw YAML**, and serves a **database download**.
- Ensures the library and download **directories exist** on startup.

## How it works

```
routes.go    → registers /settings + /config routes
handlers.go  → HTTP layer: render settings, apply form edits, expose config
manager.go   → the thread-safe Manager: load, validate, save, env-var expansion
config.go    → the Config struct hierarchy (the YAML schema)
default.go   → the built-in default configuration
telegram.go  → Telegram-related config helpers
```

### The `Manager`

`Manager` (`manager.go`) wraps a `*Config` behind a `sync.RWMutex`:

- **`NewManager(path)`** — if the file is missing, writes `defaultConfig` to disk and
  uses it; otherwise loads and validates the file. Either way it calls
  `EnsureDirectories()` to create the library and download folders.
- **`Get()`** returns the current config (read-locked); **`Update()`** swaps in a new
  config; **`Save()`** re-encodes the in-memory config to YAML on disk.
- **`GetEnabledMetadataProviders()` / `GetEnabledLyricsProviders()`** are convenience
  reads used by the metadata and lyrics features.
- **`GetYAML()`** returns the raw on-disk file contents for display.

### Environment-variable references

`loadConfig` parses the YAML into a node tree and walks it with
`processEnvVarNodes` *before* decoding into the struct. Any scalar tagged
`!env_var` is replaced with the value of the named environment variable — and the app
**fails to start** if that variable is unset or empty. This keeps secrets (API keys,
tokens) out of the committed config:

```yaml
telegram:
  token: !env_var TELEGRAM_TOKEN
metadata:
  providers:
    discogs:
      secret: !env_var DISCOGS_SECRET
```

After expansion the struct is validated with `go-playground/validator`; `libraryPath`,
`downloadPath`, and `database.path` are required.

### Editing settings from the UI

`UpdateSettings` (`handlers.go`) rebuilds a `Config` from form fields, but
**deliberately preserves** a few areas that should not change at runtime — database
settings, downloader plugins, artwork settings, lyrics providers, server settings,
the auto-start-watcher flag, and webhook config — copying them from the current
config. It then calls `Manager.Update()` and `Manager.Save()`. A save failure is
logged as a warning rather than a hard error, since the filesystem may be read-only
in containerized deployments (the in-memory update still takes effect).

## The config schema

The top-level `Config` struct (`config.go`) maps directly to `config.yaml`:

| Block | Purpose |
|-------|---------|
| `libraryPath` / `downloadPath` | Root directories (both required) |
| `database.path` | SQLite file location (required) |
| `server` | Listen `port`, `show_routes` |
| `import` | Move vs. copy, duplicate policy, queue behaviour, path templates, `allow_missing_metadata`, watcher |
| `metadata.providers` | Tagging providers (`musicbrainz`, `discogs`, `deezer`, `acoustid`) with `enabled` + optional `secret` |
| `lyrics.providers` | Lyrics providers with `enabled` + `prefer_synced` |
| `downloaders` | Plugin list + embedded-artwork settings |
| `logger` | `enabled`, `level`, `format`, `htmx_debug` |
| `jobs` | Per-job logging + webhooks |
| `telegram` | Bot `enabled`, `token`, `allowedUsers`, `bot_handle` |

See [importing](./importing.md) and [paths](../paths.md) for the `import` block,
[plugins](../plugins.md) for `downloaders`, and [jobs](./jobs.md) for the `jobs` block.

## Endpoints

Registered in `src/features/config/routes.go`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/settings` | Render the Settings section | Section |
| GET | `/config/form` | Render the settings form partial | Partial |
| PUT | `/settings` | Apply a settings form submission | Toast |
| GET | `/config` | Config as JSON (or raw YAML with `?fmt=yaml`) | JSON / YAML |
| GET | `/config/database/download` | Download the SQLite database file | Resource |

## Related

- [Importing](./importing.md), [Metadata](./metadata.md), [Lyrics](./lyrics.md), [Jobs](./jobs.md) — all read their settings from the `Manager`.
- [Paths](../paths.md) — the `import.paths` template syntax.
- [Plugins](../plugins.md) — downloader and provider configuration.
- [Hosting](./hosting.md) — wires the `Manager` into the server and every feature.
