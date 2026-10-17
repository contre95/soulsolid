---
weight: 60
title: "Reorganize"
description: "Rewriting on-disk file paths to match the configured library layout."
icon: "folder_managed"
draft: false
toc: true
---

The **reorganize** feature brings the files already in your library into line with
the configured path template. When you change your naming scheme (or import files
that were placed before a template change), the audio files on disk no longer match
where the library *would* put them. Reorganize walks every track, computes its
desired path from the current configuration, and moves any file that is in the wrong
place — updating the database to point at the new location.

## What it does

- Runs a library-wide **`analyze_reorganize`** job that relocates misplaced files.
- Computes each track's **target path** from the live path configuration (the same
  logic used at import time — see [paths](../paths.md)).
- Optionally rewrites paths to be **FAT32-safe**, stripping characters that are
  illegal on FAT/exFAT volumes and resolving any resulting name collisions.
- Skips tracks that are **already correct** or whose source file is **missing**.
- Keeps the database in sync by updating each moved track's stored `Path`.

## How it works

This is one of the analyze-section features. It is intentionally thin — a handler,
a service that starts a job, and the job task that does the work:

```
routes.go    → registers /analyze/reorganize and /analyze/files
handlers.go  → HTTP layer: parse the fat32_safe flag, start the job
service.go   → StartReorganizeAnalysis → JobService.StartJob("analyze_reorganize")
job.go       → the background job task that moves files
sanitize.go  → thin wrappers over infra/files FAT32 helpers
```

- **`Service`** (`service.go`) is wired with the `music.Library`, the
  `music.FileManager` (the path/organizer in `infra/files`), the `config.Manager`,
  and the `music.JobService`. Its only job is to launch the background job with the
  `fat32_safe` flag in the job metadata.

### The `analyze_reorganize` job

The job task (`job.go`) iterates the library in **batches of 100** and, for each
track:

1. Asks the `FileManager` for the track's **desired path**
   (`GetLibraryPath`) from the current config.
2. **Skips** the track if its source file no longer exists on disk.
3. Cleans both paths and, when `fat32_safe` is set, runs the desired path through
   `sanitizeFAT32Path` and then `resolvePathConflict` to avoid clobbering an
   existing file.
4. If the current and desired paths are **equal**, the track is counted as
   `skipped` (already correct) and left alone.
5. Otherwise it **moves the file** via `FileManager.MoveTrackFile`, then updates the
   track's `Path` in the database.

The job honors cancellation between batches and tracks (`ctx.Done()`), reports
progress 0→100%, and tallies `moved`, `skipped`, and `errors`. Individual failures
(failed path computation, failed move, failed DB update) are logged and counted but
do not abort the run.

### FAT32 safety

`sanitize.go` delegates to `infra/files`:

- `SanitizeFAT32Path` removes/replaces characters forbidden on FAT32/exFAT
  (`" * : < > ? \ |` and friends) from each path segment.
- `ResolvePathConflict` appends a disambiguating suffix when the sanitized path would
  collide with a file that already exists.

This mode is useful when syncing the library to a USB stick, SD card, or other
FAT-formatted device.

## Endpoints

Registered in `src/features/reorganize/routes.go`.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/analyze/files` | Render the File-Paths section (shows current path config) | Section |
| POST | `/analyze/reorganize` | Start the `analyze_reorganize` job | Toast (job) |

### Form parameters for `POST /analyze/reorganize`

| Field | Meaning |
|-------|---------|
| `fat32_safe` | `true` → also strip FAT32-illegal characters and resolve collisions |

## Configuration

Reorganize has no config block of its own. It reuses the **path templates** that
govern where imported files are placed. Adjust those (and preview them) under the
File-Paths section; the same templates drive both import placement and
reorganization. See [paths](../paths.md) for the template syntax and tokens.

## Related

- [Paths](../paths.md) — the path-template syntax that defines the target layout.
- [Importing](./importing.md) — applies the same path logic when files first land.
- [Jobs](./jobs.md) — runs and tracks the `analyze_reorganize` job.
- [Library](./library.md) — the collection whose files get relocated.
