---
weight: 80
title: "Streaming"
description: "Serving audio files for in-browser playback with path-traversal protection."
icon: "graphic_eq"
draft: false
toc: true
---

The **streaming** feature is a single, security-focused endpoint that serves audio
files from disk so the browser's audio player can play them. It is intentionally
minimal — there is no transcoding and no session state — but it carefully validates
every requested path to ensure only files inside the configured library or download
directories, with a recognised audio extension, are ever served.

## What it does

- Serves an audio file given its on-disk `path` via `GET /stream?path=...`.
- **Guards against path traversal** — only files under the configured `library_path`
  or `download_path` can be read, even via `../` sequences or symlinks.
- Restricts serving to **known audio MIME types** and sets the correct
  `Content-Type`.
- Advertises `Accept-Ranges: bytes` so browsers can seek and range-request audio.

## How it works

A two-file feature: a thin handler and a service that does validation.

```
routes.go    → registers GET /stream
handlers.go  → HTTP layer: decode the path param, set headers, SendFile
service.go   → path validation + MIME resolution
```

- **`Handler.Stream`** reads the `path` query parameter, URL-unescapes it, and asks
  the service to validate it. On success it sets `Content-Type` and
  `Accept-Ranges: bytes`, then serves the file with Fiber's `SendFile`. Any
  validation error is collapsed into a `404 track not found` so the endpoint never
  leaks whether a given path exists outside the allowed roots.

### Path-traversal protection

The heart of the feature is `containedIn` in `service.go`:

1. Both the candidate path and the base directory are run through
   `filepath.EvalSymlinks` (after `filepath.Clean`), which **fully resolves symlinks**.
2. The resolved candidate must either equal the resolved base or sit under it (prefix
   check including the path separator).

Resolving symlinks *before* the prefix check is what makes this robust: neither
`../..` sequences nor a symlink planted inside the allowed directory can be used to
escape it. `Service.Stream` tries this against both the library path and the
download path and accepts the file only if it falls inside one of them.

### Allowed formats

The service maps file extensions to MIME types via the `audioMIME` table. Only files
whose extension is present are served:

| Extension | MIME type |
|-----------|-----------|
| `.mp3` | `audio/mpeg` |
| `.flac` | `audio/flac` |
| `.wav` | `audio/wav` |
| `.aac` | `audio/aac` |
| `.m4a` | `audio/mp4` |
| `.ogg`, `.opus` | `audio/ogg` |
| `.wma` | `audio/x-ms-wma` |

> Note: the source flags `.wav`, `.aac`, `.m4a`, `.ogg`, `.opus`, and `.wma` as not
> yet fully supported elsewhere in Soulsolid; `.mp3` and `.flac` are the primary
> formats.

### The `SendFile` encoding quirk

Fiber's `SendFile` feeds the path to fasthttp as a request URI, so characters such as
`?` or `#` in a filename would be parsed as a query string or fragment and truncate
the path — yielding a spurious 404. To avoid this the handler percent-encodes the
resolved path (`(&url.URL{Path: resolved}).EscapedPath()`) so it round-trips through
fasthttp's URI decoding intact.

## Endpoints

Registered in `src/features/streaming/routes.go`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/stream?path=<url-encoded path>` | Stream an audio file for playback | Audio bytes / 404 / 400 |

Possible responses:

- `200` with the audio bytes and the matching `Content-Type`.
- `400 missing path` / `400 invalid path` — no path, or an undecodable path.
- `404 track not found` — path outside the allowed roots or unsupported file type.

## Configuration

Streaming has no config block of its own; it reads two paths from the global config
through the `config.Manager`:

```yaml
library_path: /app/library     # allowed root #1
download_path: /app/downloads  # allowed root #2
```

Only files under these two directories are ever served.

## Related

- [Library](./library.md) — exposes track paths that the player passes to `/stream`.
- [Playlists](./playlists.md) — playback of playlist tracks goes through this endpoint.
- [Downloading](./downloading.md) — files in the download directory are also streamable.
