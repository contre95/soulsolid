---
weight: 20
title: "Downloading"
description: "Searching external sources and fetching tracks, albums, artists, and playlists via plugins."
icon: "download"
draft: false
toc: true
---

The **downloading** feature lets Soulsolid search external music sources and pull
down tracks, albums, artists, and playlists. Soulsolid ships with **no built-in
source**: every source is provided by a **downloader plugin** loaded at startup, so
the set of available providers (and their capabilities) depends entirely on your
configuration.

Downloads land in the configured `download_path`. They are **not** added to the
library directly — instead the [importing](./importing.md) feature picks them up
(optionally via the directory watcher) and organizes them into the library.

## What it does

- **Searches** an external source for albums, tracks, artists, or by direct **link**
  (URL paste), rendering provider-specific result partials.
- **Starts download jobs** for a single track, an album, an artist's discography, an
  arbitrary set of tracks, or a named playlist.
- **Browses** an album's track list and a source's **chart** (top tracks).
- Reports the connected **user info**, **status** (credential validity), and
  **capabilities** of each configured downloader.

## How it works

### Plugin architecture

The core abstraction is the `Downloader` interface (`downloader.go`). Any plugin must
implement search, navigation, download, and info methods:

```go
type Downloader interface {
    SearchAlbums / SearchTracks / SearchArtists / SearchLinks
    GetAlbumTracks / GetArtistAlbums / GetChartTracks
    DownloadTrack / DownloadAlbum / DownloadArtist / DownloadLink
    GetUserInfo() *UserInfo
    GetStatus() DownloaderStatus
    Name() string
    Capabilities() DownloaderCapabilities
}
```

Plugins are compiled Go **`-buildmode=plugin` `.so` files** that export a
`NewDownloader(config map[string]interface{}) (Downloader, error)` symbol. The
`PluginManager` (`plugins.go`) loads them at startup from one of three sources per
plugin config entry:

1. **`url:`** — clones a git repo (`--depth 1`), adds a `replace` directive pointing
   at the local soulsolid module, runs `go mod tidy`, and builds the `.so` in a temp
   dir. This requires the Go toolchain to be present at runtime.
2. **`path:` as an `http(s)://` URL** — downloads a prebuilt `.so` to a temp file.
3. **`path:` as a local file** — opens the `.so` directly.

Each loaded plugin is registered under its config `name`. `GetDownloader(name)` looks
it up; missing names produce a "downloader not found" error. The manager is
mutex-guarded so it is safe for concurrent request handling.

> See the dedicated [plugins guide](../plugins.md) for writing and packaging a plugin.

### Capabilities & status

A downloader advertises `DownloaderCapabilities` (search / artist search / direct
links / chart tracks). The UI uses these to show or hide features — e.g. the chart
view renders a "not supported" state when `SupportsChartTracks` is false.

`GetStatus()` returns a `DownloaderStatus` with one of `disabled`,
`invalid_credentials`, or `valid`, surfaced in the user-info panel so you can tell at
a glance whether credentials are working.

### Search flow

`Handler.Search` is the general entry point and dispatches on a `type` field
(`album`, `track`, `artist`, `link`). The dedicated `/search/albums` and
`/search/tracks` routes are thin shortcuts. Search limits are clamped by the service
to **1–100** (default 20). The `link` type accepts a pasted URL and returns a
`LinkResult` tagged `track`/`album`/`playlist`/`artist`, rendered into the matching
partial; for non-HTMX clients it returns the raw `LinkResult` as JSON.

### Download flow (jobs)

Downloads are **asynchronous**. Each download handler validates input, then calls the
service, which starts a background [job](./jobs.md) via `JobService.StartJob(...)` and
immediately returns a **job ID** (HTTP `202`, rendered as a "download started" toast).
The active downloader is chosen by the `?downloader=` query param (default `dummy`).

The actual work runs in `DownloadJobTask.Execute` (`download_job.go`), keyed by the
job's `type` metadata:

- Ensures `download_path` exists (`MkdirAll`).
- Dispatches to the per-type executor (`track`/`album`/`artist`/`tracks`/`playlist`).
- Streams progress through the job's `progressUpdater` — the download phase is mapped
  to the 25–75% band of overall progress, with a callback reporting MB downloaded.
- Writes tags/artwork via the injected `TagWriter` (artwork embedding is controlled by
  `downloaders.artwork.embedded`).
- Filenames are sanitized (`Sanitize`) to strip filesystem-unsafe characters.

Five job handler types are registered in `main.go` (`download_track`,
`download_album`, `download_artist`, `download_tracks`, `download_playlist`), all
backed by the same `DownloadJobTask`.

## Endpoints

Registered in `src/features/downloading/routes.go` under `/downloads`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/downloads` | Render the Download section | Section |
| POST | `/downloads/search` | General search (`type=album\|track\|artist\|link`) | Partial / JSON |
| POST | `/downloads/search/albums` | Album search shortcut | Partial |
| POST | `/downloads/search/tracks` | Track search shortcut | Partial |
| GET | `/downloads/album/:albumId/tracks` | List an album's tracks | Partial |
| POST | `/downloads/track` | Start a track download | Toast Job (202) |
| POST | `/downloads/album` | Start an album download | Toast Job (202) |
| POST | `/downloads/artist` | Start a full-artist download | Toast Job (202) |
| POST | `/downloads/tracks` | Download multiple tracks (CSV `trackIds`) | Toast Job (202) |
| POST | `/downloads/playlist` | Download a playlist (`trackIds` + `playlistName`) | Toast Job (202) |
| GET | `/downloads/capabilities` | Downloader capabilities | JSON |
| GET | `/downloads/user/info` | Connected user info + statuses | Partial |
| GET | `/downloads/chart/tracks` | Source chart / top tracks | Partial |

Most routes accept a `?downloader=<name>` query param selecting which plugin to use.
Search bodies use the `SearchRequest` form/JSON shape (`query`, `type`, `limit`,
`downloader`).

## Configuration

```yaml
downloaders:
  plugins:
    - name: dummy
      url: https://github.com/contre95/soulsolid-dummy-plugin  # build from git, OR
      # path: ../soulsolid-dummy-plugin/plugin.so              # local/remote .so
      icon: https://demo2.contre.io/img/galaxy.png
      config: {}        # plugin-specific config passed to NewDownloader
  artwork:
    embedded:
      enabled: true     # embed cover art into downloaded files
      size: 1000        # max artwork dimension
      quality: 85       # JPEG quality
download_path: /app/downloads   # where downloaded files are written
```

The first configured plugin is used as the default selection when the Download section
is opened without an explicit `?downloader=`.

## Related

- [Plugins](../plugins.md) — building and packaging downloader plugins.
- [Jobs](./jobs.md) — the async engine that runs every download.
- [Importing](./importing.md) — picks up downloaded files into the library.
- [API reference](../api.md) — full route table.
