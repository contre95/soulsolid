---
weight: 30
title: "Importing"
description: "Scanning directories, fingerprinting, duplicate handling, and the manual-review queue."
icon: "drive_folder_upload"
draft: false
toc: true
---

The **importing** feature ingests audio files from a local directory into the library:
reading tags, computing acoustic fingerprints, detecting duplicates and missing
metadata, organizing files into the library tree, and surfacing anything that needs a
human decision in a **review queue**.

> This page focuses on how the feature is built and its endpoints. For the full
> configuration reference (path templates, duplicate policy, FAT32 options) see the
> [importing configuration guide](../importing.md) and [path templates](../paths.md).

## What it does

- **Imports a directory** recursively as a background [job](./jobs.md), processing only
  `.mp3` and `.flac` files.
- **Detects duplicates** via acoustic fingerprinting and applies the configured policy
  (`queue` / `skip` / `replace`).
- **Holds problem items in a queue** for manual review: duplicates, tracks with missing
  required metadata, failed imports, and explicit manual-review items.
- Lets you **act on queued items** individually or in **bulk by artist/album**:
  import, replace, skip/cancel, or delete.
- Optionally **watches** the download path and auto-imports new files.
- **Prunes** the download path (deletes leftover audio + clears the queue).

## How it works

### Layers and collaborators

The service (`service.go`) is wired in `main.go` with a rich set of collaborators:

| Collaborator | Role |
|--------------|------|
| `music.Library` | Persistence (find/add/update tracks, artists, albums) |
| `TagReader` | Reads tags + embedded artwork from files |
| `FingerprintProvider` | Computes acoustic fingerprints (chromaprint) |
| `music.FileManager` | Copies/moves/deletes files using path templates |
| `config.Manager` | Live import config |
| `music.JobService` | Runs the directory import asynchronously |
| `music.Queue` | In-memory review queue |
| `Watcher` | Filesystem watcher over the download path |

### Directory import (the job)

`ImportDirectory` starts a `directory_import` job; the heavy lifting lives in
`directory_job.go` (registered as a job handler in `main.go`). For each supported file
the importer:

1. Reads tags (and computes a fingerprint) to build a `music.Track`.
2. Evaluates whether the track is a **duplicate** (fingerprint/ID match) and whether it
   is **missing required metadata** (per the `allow_missing_metadata` config).
3. Decides to import directly or to **enqueue** the item with one or more **status
   types**, then continues to the next file.

`ImportStats` (errors / albums / tracks / artists imported / skipped / queued) is
accumulated and reported on the job.

### Importing a single track

`importTrack` (used by both the directory job and manual queue processing) is the core
write path:

1. **Fill defaults** — `EnsureMetadataDefaults` fills any *permitted* missing fields
   (artist/album/title/year/genre) with fallbacks **before** path resolution, so they
   feed the destination path template.
2. **Organize the file** — move or copy into the library via the `FileManager`
   (`Import.Move` decides which), producing the new on-disk path.
3. **Resolve relations** — `populateTrackArtistsAndAlbum` find-or-creates the artist(s)
   and album rows so the track links to real DB entities.
4. **Validate** then **persist**, with two idempotency guards:
   - if a track already exists at the same **path**, skip;
   - if a track with the same **ID** exists at a different path, update it in place and
     delete the old file (handles re-imports / relocations).

`replaceTrack` is the duplicate-resolution variant: it organizes the new file, copies
the new metadata onto the existing DB row, updates it, and removes the old file.

### The review queue & status types

Queued items (`music.QueueItem`) can carry **several status types at once**
(`HasType(...)`):

- **Duplicate** — matches an existing library track.
- **MissingMetadata** — missing a required field that isn't allowed to be defaulted.
- **FailedImport** — the import attempt errored.
- **ManualReview** — explicitly queued for review (e.g. `always_queue`).

The handler's `convertQueueItem` derives **which buttons are available** from these
types, encoding the rule *"a track missing required metadata cannot enter the library
until it's fixed"*:

- **Replace** shows only for duplicates, and is disabled while metadata is missing or
  the import failed (never overwrite the library with an incomplete track).
- **Import** shows for manual-review / missing-metadata items (never duplicates/failed)
  and stays disabled until the missing metadata is fixed.
- The cancel button is labeled **Skip** for duplicates/failed items, **Cancel**
  otherwise.

`ProcessQueueItem` enforces the same rules server-side, rejecting disallowed actions
even if a client bypasses the UI.

### Grouped (bulk) actions

Items can be grouped **by artist** or **by album**. `ProcessQueueGroup` applies an
action to each item in a group, filtering by type so it does the right thing:
`import` skips duplicates (use `replace` for those) and `replace` only touches
duplicates. If every attempted item fails, the group operation reports failure rather
than a misleading success.

### The watcher

When `auto_start_watcher` is on (or toggled via the UI), the `Watcher` monitors the
download path. On a file-create event, `handleFileEvent` **waits for any running jobs
to finish** (polling up to 5 minutes) and then kicks off a directory import of the
download path. This prevents the watcher from racing an in-progress download/import.

### HTMX triggers

Mutating handlers set `HX-Trigger` headers (e.g. `queueUpdated`,
`refreshImportQueueBadge`, `activateIndividualGrouping`, `watcherStatusChanged`) so the
frontend can refresh the queue, badge counts, and grouping views reactively.

## Endpoints

Registered in `src/features/importing/routes.go` under `/import`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/import` | Render the Import section | Section |
| GET | `/import/directory/form` | Directory import form (with default path) | Partial |
| POST | `/import/directory` | Start a directory import job | Toast Job (202) |
| GET | `/import/queue/items` | Render queue items (oldest first, capped at 10) | Partial |
| GET | `/import/queue/items/grouped` | Queue items grouped (`?type=artist\|album`) | Partial |
| GET | `/import/queue/header` | Queue header with count | Partial |
| GET | `/import/queue/count` | Queue count, formatted `(N)` | Text |
| GET | `/import/queue/:id/artwork` | Embedded artwork for a queued track | Binary |
| POST | `/import/queue/:id/:action` | Item action: `import`/`replace`/`cancel`/`delete` | Toast |
| POST | `/import/queue/group/:groupType/:groupKey/:action` | Bulk group action | Toast |
| POST | `/import/queue/clear` | Clear the entire queue | Toast |
| POST | `/import/prune/download-path` | Delete download-path audio + clear queue | Toast |
| POST | `/import/watcher/toggle` | Start/stop the watcher (`action=start\|stop`) | Toast |
| GET | `/import/watcher/status` | Watcher status badge | Partial |
| GET | `/import/watcher/toggle-state` | Watcher toggle input (checked state) | Partial |

## Configuration

```yaml
import:
  move: false              # copy (false) or move (true) source files into the library
  always_queue: false      # queue every track for manual review
  duplicates: queue        # queue | skip | replace
  allow_missing_metadata:  # per-field: true = fill with default, false = send to queue
    artist: false
    album: false
    title: false
    year: false
    genre: false
  auto_start_watcher: false
  paths:
    default_path: '%asciify{$albumartist}/%asciify{$album}/%asciify{$track $title}'
    fat32_safe: false
download_path: /app/downloads   # watched + pruned location
```

Only `.mp3` and `.flac` files are processed (`supportedExtensions`).

## Telegram

The importing service is injected into the Telegram bot, which can trigger imports from
chat. See [hosting](./hosting.md).

## Related

- [Importing configuration & path templates](../importing.md), [paths](../paths.md)
- [Jobs](./jobs.md) — runs the directory import.
- [Metadata](./metadata.md) — fixing missing tags before importing a queued item.
- [Downloading](./downloading.md) — produces the files the watcher picks up.
