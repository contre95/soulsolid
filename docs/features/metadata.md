---
weight: 40
title: "Metadata / Tagging"
description: "Editing track tags, fetching metadata from providers, fingerprinting, and AcoustID analysis."
icon: "sell"
draft: false
toc: true
---

The **metadata** feature (mounted under `/tag` and `/analyze`) is the tag editor and
metadata-enrichment engine. It lets you view and edit a track's tags, pull candidate
metadata from external providers (MusicBrainz, Discogs, Deezer), compute acoustic
fingerprints, and bulk-identify the whole library via **AcoustID**.

## What it does

- Renders a **tag editor** for a track, reading the latest tags **from the file** and
  merging in database relationships (artists/albums).
- **Fetches and merges** metadata from a provider, or runs a **search** and lets you
  pick the best match from a results modal.
- **Writes tags** back to both the file and the database, creating artists/albums as
  needed.
- Computes a **chromaprint fingerprint** and looks up its **AcoustID** for a track.
- Runs a library-wide **AcoustID analysis** job to fingerprint/identify every track.
- Serves a track's **embedded artwork**.

## How it works

### Providers

External lookups go through the `MetadataProvider` interface (`metadata.go`):

```go
type MetadataProvider interface {
    SearchTracks(ctx, SearchParams) ([]*music.Track, error)
    Name() string
    IsEnabled() bool
}
```

Three providers are wired in `main.go` and injected as a map keyed by name:
`musicbrainz`, `discogs`, `deezer` (each only active if enabled in config). `SearchParams`
carries the hints used to search — album-artist, album, title, year, and AcoustID.

### Reading tags (file as source of truth)

`GetTrackFileTags` reads the **current tags from the file** via the `TagReader`, then
overlays them onto the database track so the editor always shows what's actually on
disk (title, ISRC, lyrics, audio properties) while keeping DB relationships. It also
runs `matchArtistsWithDatabase` so artist dropdowns resolve to real IDs. The editor can
also load directly from the DB with `?source=db`.

The handler does a lot of **dropdown reconciliation**: any artist/album referenced by
the track but absent from the main lists is appended, and entities without a DB ID get
a temporary `temp_<name>` id so they can still be displayed and selected.

### Fetch vs. search-and-select

There are two ways to apply provider metadata:

- **Fetch** (`GET /tag/:trackId/:provider`) — searches the provider, takes the first
  result, **merges** it onto the current track (`MergeFetchedData`, preserving
  file-specific fields), matches the fetched album against existing albums, and
  re-renders the editor pre-filled.
- **Search & select** — `GET /tag/:trackId/search/:provider` returns a modal of
  candidates; `GET /tag/:trackId/select/:provider?index=N` applies the chosen one,
  find-or-creating its artists and album before merging.

Each provider has a color theme (`getProviderColors`) so the pre-filled fields are
visually attributed to their source.

### Writing tags

`UpdateTags` (`POST /tag/:trackId`) collects form fields into a `map[string]string`
(title, artist IDs, album, year, genre, track/disc number, ISRC, composer, lyrics, BPM,
gain, title version, source…). An **instrumental** checkbox clears lyrics and sets
`has_lyrics=false`. The service builds an updated track, creates the album if it's new,
then **writes tags to the file** (`TagWriter`) and persists to the database.

### Fingerprinting & AcoustID

`AddChromaprintAndAcoustID` computes a track's chromaprint and resolves its AcoustID via
the `ChromaprintAcoustID` interface (implemented by `providers.AcoustIDService`):

```go
GenerateChromaprint(ctx, filePath) (fingerprint, duration, error)
CompareChromaprints(cp1, cp2) (similarity, error)
LookupAcoustID(ctx, chromaprint, duration) (acoustID, error)
```

The library-wide **AcoustID job** (`acoustid_job.go`, registered as `analyze_acoustid`)
processes every track in **batches of 100**, skips tracks that already have an
`acoustid` attribute, honors cancellation between batches, reports progress, and
returns counts (processed / acoustIDs added / fingerprints added / skipped).

## Endpoints

Registered in `src/features/metadata/routes.go` under `/tag` and `/analyze`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/tag/:trackId` | Render the tag editor (`?source=file\|db`) | Section |
| POST | `/tag/:trackId` | Save tag changes (file + DB) | Toast |
| GET | `/tag/:trackId/:provider` | Fetch + merge metadata, re-render editor | Section |
| GET | `/tag/:trackId/metadata` | Enabled provider buttons | Partial / JSON |
| GET | `/tag/:trackId/search/:provider` | Search results modal | Partial |
| GET | `/tag/:trackId/select/:provider?index=N` | Apply a selected result | Section |
| GET | `/tag/:trackId/artwork` | Serve embedded artwork | Resource |
| GET | `/tag/:trackId/fingerprint` | Compute chromaprint + AcoustID | Toast |
| GET | `/tag/:trackId/fingerprint/view` | View stored fingerprint string | Text |
| POST | `/analyze/acoustid` | Start library-wide AcoustID job | Toast Job (202) |
| GET | `/analyze/metadata` | Render the metadata-analysis section | Section |

> The `:provider` and `:trackId` routes share the `/tag/:trackId/*` space — ordering in
> `routes.go` matters so specific paths (`/metadata`, `/artwork`, `/search/...`) match
> before the catch-all `/tag/:trackId/:provider`.

## Configuration

```yaml
metadata:
  providers:
    acoustid:
      enabled: true
      # secret: !env_var ACOUSTID_CLIENT_KEY   # acoustid.org application key
    deezer:
      enabled: true
    discogs:
      enabled: true
      # secret: !env_var DISCOGS_API_KEY
    musicbrainz:
      enabled: true
```

A disabled provider is omitted from the editor's provider buttons. AcoustID lookups
require a client key to return identifications.

## Related

- [Lyrics](./lyrics.md) — sibling `/tag/:trackId/lyrics` routes and a similar analyze job.
- [Importing](./importing.md) — fixing missing metadata on queued items uses this editor.
- [Jobs](./jobs.md) — runs the AcoustID analysis.
- [Library search](./library.md) — the `has_acoustid` filter relies on this data.
