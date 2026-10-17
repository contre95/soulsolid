---
weight: 10
title: "Library"
description: "Browsing, searching, and managing the music collection."
icon: "library_music"
draft: false
toc: true
---

The **library** feature is the heart of Soulsolid. It is the read/search/delete
layer over the music collection that has already been imported. Everything other
features produce (imports, downloads, tagging, lyrics) ultimately lands in the
library, and this feature is responsible for exposing it: paginated browsing,
unified search, per-entity detail panels, storage statistics, a filesystem tree
view, and cascading deletes.

## What it does

- Browses the collection as a paginated **track** list (server-side pagination).
- Provides a single **unified search** box that matches across albums, artists, and
  tracks at once — showing **albums and artists before tracks** — with optional
  filters (genre, AcoustID presence, lyrics state).
- Renders **detail panels** — a floating track-overview panel and per-entity JSON
  endpoints.
- Reports **collection statistics**: artist/album/track counts and total storage
  size on disk.
- Exposes a **file-tree** view of the library and download directories.
- Performs **cascading deletes** of tracks, albums, and artists — removing both the
  database rows and the underlying files on disk.

## How it works

The feature follows the same three-layer shape used across the codebase:

```
routes.go      → registers HTTP routes on the Fiber app
handlers.go    → HTTP layer: parse params, call the service, render via respond.*
service.go     → domain layer: business logic, wraps the library + file manager
```

- **`Handler`** (`handlers.go`) is the HTTP boundary. It reads query params and path
  params, calls the service, and hands the result to the `respond` package, which
  performs content negotiation (HTMX fragment vs. JSON — see the [API reference](../api.md)).
- **`Service`** (`service.go`) is the domain layer. It is constructed with three
  collaborators wired in `main.go`:
  - `library.Library` — the persistence interface, implemented by the SQLite store
    (`infra/database`). All counts, fetches, and searches delegate here.
  - `config.Manager` — supplies live config (library path, download path).
  - `library.FileManager` — the file organizer (`infra/files`), used to delete the
    actual audio files from disk during cascading deletes.

### Pagination

`handlers.go` defines a small `Pagination` helper (`NewPagination(page, limit,
totalCount)`) that computes total pages and next/prev links. Browsing endpoints take
`page` and `limit` query params (`limit` defaults to 50) and convert them to a
SQL `LIMIT/OFFSET` via the service's `*Paginated` methods.

### Unified search

`GetUnifiedSearch` is the most involved handler. It produces a single combined,
paginated list of `SearchResult` items (each tagged `artist`, `album`, or `track`)
and has two modes:

- **Browse-all** (no query, no filters): only **tracks** are listed, paginated by
  offset/limit. Albums and artists are not included in the browse view.
- **Search/filter**: when a text query is present, results are ordered **albums,
  then artists, then tracks**. Albums and artists are matched by name/title (capped
  at 20 each) and shown first; tracks are matched with a single `TrackFilter` query
  that OR-matches title, artist name, and album title and AND-combines the active
  filters:
  - `query` → `TextSearch`
  - `genre` → exact genre match
  - `has_acoustid=true|false` → presence of an AcoustID fingerprint
  - `lyrics_filter` / `lyrics_text` → lyrics state and full-text lyrics search

  Track results are offset relative to the album/artist results that precede them so
  pagination stays correct across the combined list.

Album search uses a dedicated lightweight `SearchAlbums` query (single JOIN, no N+1)
to keep it fast.

### Cascading deletes

`DeleteTrack`, `DeleteAlbum`, and `DeleteArtist` follow a **DB-first, files-second**
strategy:

1. Fetch the affected tracks (to capture their file paths) before deletion.
2. Delete the rows from the database (artist/album deletes cascade to their tracks).
3. Delete each track's file from disk via the `FileManager`.

File deletion failures are logged as warnings but do **not** fail the request — the
database is treated as the source of truth, and orphaned files are considered a
secondary concern.

### Statistics & file tree

- `GetStorageSize` walks the library path with `filepath.Walk`, summing file sizes,
  and formats the result (B/KB/MB/GB/TB).
- `GetLibraryFileTree` shells out to the system `tree` command against either the
  library or downloads path (selected by the `folder` query param). If `tree` is not
  installed, the error surfaces in a toast.

## Endpoints

All routes are registered in `src/features/library/routes.go` under the `/library`
group (plus the top-level section route).

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/library` | Render the Library section (with search form data) | Section |
| GET | `/library/table` | Render the tabbed library table partial | Partial |
| GET | `/library/search` | Unified search across artists/albums/tracks | Partial |
| GET | `/library/tracks/:trackId/overview` | Floating track-overview panel | Partial |
| GET | `/library/artists/count` | Number of artists | Text |
| GET | `/library/albums/count` | Number of albums | Text |
| GET | `/library/tracks/count` | Number of tracks | Text |
| GET | `/library/storage/size` | Total library size on disk (formatted) | Text |
| GET | `/library/artists/:id` | Single artist | JSON |
| GET | `/library/albums/:id` | Single album | JSON |
| GET | `/library/tracks/:id` | Single track | JSON |
| GET | `/library/tree` | Filesystem tree (`?folder=library\|downloads`) | Text |
| DELETE | `/library/tracks/:trackId` | Delete a track (DB + file) | Toast |
| DELETE | `/library/albums/:albumId` | Delete an album and its tracks | Toast |
| DELETE | `/library/artists/:artistId` | Delete an artist and their tracks | Toast |

### Query parameters for `/library/search`

| Param | Default | Meaning |
|-------|---------|---------|
| `query` | `""` | Free-text search across artists, albums, tracks |
| `page` | `1` | Page number |
| `limit` | `50` | Results per page |
| `genre` | `""` | Filter tracks by exact genre |
| `has_acoustid` | `""` | `true`/`false` — filter tracks by AcoustID presence |
| `lyrics_filter` | `""` | Filter by lyrics state |
| `lyrics_text` | `""` | Full-text search within lyrics |

> The `/library/tracks/:id/lyrics` route also exists but is registered by the
> [lyrics](./lyrics.md) feature, not here.

## Configuration

The library feature has no dedicated config block. It reads two paths from the global
config via the `config.Manager`:

```yaml
library_path: /app/library    # root scanned for storage size & file tree
download_path: /app/downloads  # used by the file-tree "downloads" view
database:
  path: /data/library.db       # SQLite store backing all queries
```

## Telegram

The library service is also injected into the Telegram bot (`hosting.NewTelegramBot`),
which exposes a subset of browse/search operations over chat. See
[hosting](./hosting.md) for the bot wiring.

## Related

- [Importing](./importing.md) and [Downloading](./downloading.md) populate the library.
- [Metadata](./metadata.md) and [Lyrics](./lyrics.md) enrich existing library tracks.
- [Metrics](./metrics.md) aggregates the same data into charts.
- [API reference](../api.md) — full route table and response-type semantics.
