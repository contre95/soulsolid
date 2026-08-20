---
weight: 50
title: "Lyrics"
description: "Fetching lyrics from providers, bulk analysis, and a review queue for conflicts."
icon: "lyrics"
draft: false
toc: true
---

The **lyrics** feature fetches song lyrics from external providers and writes them
into both the track's file tags and the database. It works in two modes: an
**on-demand** mode in the tag editor (pick a provider, pull lyrics for one track)
and a **bulk analysis** mode that walks the entire library as a background job.
Because overwriting human-curated lyrics is destructive, anything ambiguous is sent
to a **review queue** instead of being applied automatically.

## What it does

- Lists the configured **lyrics providers** and their enabled state.
- Fetches lyrics for a **single track** from a chosen provider (tag-editor flow).
- Runs a **library-wide analysis job** (`analyze_lyrics`) that attempts to add
  lyrics to every track.
- Routes conflicts and failures into a **review queue** with three item types
  (`existing_lyrics`, `lyric_404`, `failed_lyrics`) so the user decides what to do.
- Supports **grouped queue processing** (by artist or album) and bulk actions.
- Writes lyrics to the audio file tags **and** the SQLite store, keeping the
  `has_lyrics` flag consistent.

## How it works

The feature follows the standard three-layer shape, plus a job task and a queue:

```
routes.go      → registers HTTP routes (/tag, /library, /lyrics/queue, /analyze)
handlers.go    → HTTP layer: params, HTMX/JSON negotiation, queue view models
service.go     → domain logic: fetch, compare, persist, queue management
lyrics.go      → LyricsProvider interface
lyrics_job.go  → the analyze_lyrics background job task
queue.go       → queue item type constants
```

- **`Service`** (`service.go`) is wired with a `TagWriter`/`TagReader` (file tags),
  the `music.Library` (SQLite store), the map of `LyricsProvider`s, the
  `config.Manager`, a `music.Queue`, and the `music.JobService`.
- **`LyricsProvider`** (`lyrics.go`) is the provider contract: `SearchLyrics`,
  `Name`, `DisplayName`, `IsEnabled`. Providers are registered in `main.go` and
  toggled via config.

### The `AddLyrics` decision tree

`AddLyrics` is the core routine and returns an `AddLyricsResult` describing what
happened:

1. **Instrumental guard** — if the track's `HasLyrics` flag is `false`, it is
   treated as instrumental and skipped (`LyricsSkippedInstrumental`).
2. **Search** — build `LyricsSearchParams` (title, artist, album, album artist),
   validate the provider is enabled, and query it. No result / empty result →
   `LyricsSkippedNotFound`.
3. **Conflict handling** — if the track *already* has lyrics:
   - identical to the fetched text → `LyricsSkippedIdentical` (no-op).
   - different → either **override directly** (when `overrideNoQueue` is set) or
     **enqueue** an `existing_lyrics` item for manual review (`LyricsQueued`).
4. **Apply** — otherwise write the lyrics to the file tags and persist via
   `UpdateTrack`, setting `HasLyrics = true` (`LyricsAdded`).

File-tag write failures are logged as warnings but do not abort the DB update.

### The review queue

The queue holds tracks that need a human decision. Each item carries a
`QueueItemType` and per-item metadata (e.g. the candidate `new_lyrics` and the
`provider` that produced it). The three types and their allowed actions:

| Type | Meaning | Allowed actions |
|------|---------|-----------------|
| `existing_lyrics` | Provider found lyrics that **differ** from existing ones | `override`, `keep_old` |
| `lyric_404` | Provider could not find lyrics | `no_lyrics` (mark instrumental) |
| `failed_lyrics` | Fetch failed / errored | `skip`, `edit_manual`, `no_lyrics` |

`ProcessLyricsQueueItem` validates the action against the item type, applies the
change (writing tags + DB where relevant), and removes the item. Group processing
(`ProcessLyricsQueueGroup`) fans the same action out over every item grouped under
an artist or album, continuing past individual failures.

Handlers emit `HX-Trigger` events (`lyricsQueueUpdated`, `refreshLyricsQueueBadge`,
`updateLyricsQueueCount`, and grouping-activation events) so the UI refreshes badges
and lists after each action.

### The `analyze_lyrics` job

`StartLyricsAnalysis` launches the `analyze_lyrics` job with three options carried in
the job metadata: `provider`, `skip_existing`, and `override_no_queue`. The job task
(`lyrics_job.go`):

- Aborts early if no provider is enabled.
- Iterates the library in **batches of 100** (paginated) to bound memory.
- Honors cancellation via `ctx.Done()` between batches and tracks.
- Per track, applies the skip rules (skip tracks that already have lyrics when
  `skip_existing`; skip instrumentals unless `override_no_queue`), then calls
  `AddLyrics`.
- Tallies `updated`, `skipped`, `errors`, and the number of new queue items by type,
  and reports a final summary plus progress (0→100%).

Per-track failures never fail the whole job — they are counted and logged with a
manual-fix link to the track's tag editor.

## Endpoints

Routes are registered in `src/features/lyrics/routes.go` across four groups.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/tag/:trackId/lyrics` | Provider buttons for a track (or provider list as JSON) | Partial / JSON |
| GET | `/tag/:trackId/lyrics/text/:provider` | Fetch lyrics text from a provider | Text / JSON / Toast |
| GET | `/library/tracks/:id/lyrics` | The track's stored lyrics, plain text | Text |
| GET | `/lyrics/queue/header` | Queue header partial | Partial |
| GET | `/lyrics/queue/items` | Ungrouped queue items (capped at 10) | Partial |
| GET | `/lyrics/queue/items/grouped` | Queue items grouped (`?type=artist\|album`) | Partial |
| POST | `/lyrics/queue/:id/:action` | Process a single queue item | Toast |
| POST | `/lyrics/queue/group/:groupType/:groupKey/:action` | Process a whole group | Toast |
| POST | `/lyrics/queue/clear` | Clear the entire queue | Toast |
| GET | `/lyrics/queue/count` | Queue count, formatted as `(N)` | Text |
| GET | `/lyrics/queue/:id/new_lyrics` | Candidate lyrics held in a queue item | Text |
| POST | `/analyze/lyrics` | Start the `analyze_lyrics` job | Toast (job) |
| GET | `/analyze/lyrics` | Render the lyrics-analysis section | Section |

### Form parameters for `POST /analyze/lyrics`

| Field | Meaning |
|-------|---------|
| `provider` | Lyrics provider to use (required) |
| `skip_existing_lyrics` | `true` → skip tracks that already have lyrics |
| `override_no_queue` | `true` → overwrite conflicting lyrics directly instead of queuing (ignored when skip is on) |

## Configuration

Lyrics providers are toggled in the global config under the lyrics-providers block,
read via `config.Manager.GetEnabledLyricsProviders()`. A provider is only usable when
both the config flag and the provider's own `IsEnabled()` return true. See
[plugins / providers](../plugins.md) for adding new providers.

## Related

- [Metadata / Tagging](./metadata.md) — the tag editor where on-demand lyrics fetch lives.
- [Jobs](./jobs.md) — the background-job system that runs `analyze_lyrics`.
- [Library](./library.md) — lyrics state is a search filter (`lyrics_filter`, `lyrics_text`).
- [API reference](../api.md) — response-type semantics.
