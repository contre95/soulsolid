---
weight: 100
title: "Jobs"
description: "The in-memory background-job engine: queueing, progress, logs, and webhooks."
icon: "manufacturing"
draft: false
toc: true
---

The **jobs** feature is the background-processing engine the rest of Soulsolid builds
on. Long-running operations — importing a directory, analyzing lyrics, reorganizing
files, computing metrics — are not run inline on the request; instead they are
registered as **job types** and dispatched here. The jobs service runs them one at a
time, tracks their progress and status, optionally writes per-job log files, and can
fire a webhook command when a job finishes.

## What it does

- Provides a **`JobService`** (the `music.JobService` interface) used by every
  analyze/import feature to launch work.
- Runs jobs **serially** — only one job executes at a time; the rest queue as
  `pending` and start automatically as slots free up.
- Tracks each job's **status** (`pending`, `running`, `completed`, `failed`,
  `cancelled`), **progress** (0–100), and message.
- Optionally writes a **per-job log file** and serves it raw, color-highlighted, or
  full-screen.
- Supports **cancellation** via context, plus cleanup of old/finished jobs.
- Fires an optional **webhook command** on completion, templated with job details.

## How it works

```
routes.go    → registers the /jobs routes
handlers.go  → HTTP layer: start/list/status/progress/logs/cancel + HTMX partials
service.go   → the engine: registry, scheduling, execution, webhooks
log_colors.go→ converts log text into color-highlighted HTML
```

### Tasks, handlers, and the registry

The engine separates *what to run* from *how it is run*:

- A **`Task`** (implemented by each feature, e.g. `LyricsJobTask`,
  `ReorganizeJobTask`, `MetricsCalculationTask`) defines `MetadataKeys()`,
  `Execute(ctx, job, progressUpdater)`, and `Cleanup(job)`.
- A **`TaskHandler`** wraps a task. `BaseTaskHandler` is the standard wrapper: it
  validates the required metadata keys, builds the `progressUpdater` callback, runs
  the task, and always runs `Cleanup` via `defer`.
- `RegisterHandler(jobType, handler)` registers a handler under a job-type string.
  This wiring happens in `main.go`, e.g. `analyze_lyrics`, `analyze_reorganize`,
  `metrics`, `import_directory`.

### Lifecycle and serial scheduling

`StartJob(jobType, name, metadata)`:

1. Creates a `music.Job` with a UUID, `pending` status, and the supplied metadata.
2. If job logging is enabled, creates the log directory and a dated log file
   (`YYYY-MM-DD-<id>.log`) and attaches a `slog` text logger; otherwise attaches a
   discard logger.
3. Stores the job. If **no job is currently running**, it flips to `running` and
   executes in a goroutine; otherwise it stays `pending`.

`executeJob` looks up the handler, creates a cancellable context, streams progress
updates over a channel into `UpdateJobProgress`, merges the task's returned stats
into `job.Metadata`, sets the final status, fires the webhook, and then calls
`startNextPendingJob` — which promotes the **oldest** pending job. A special case
detects "partial import" errors and records them as *completed with errors* rather
than failed.

### Cancellation, progress, and cleanup

- **Cancel** (`CancelJob`) marks the job cancelled, calls its context `CancelFunc`,
  and lets the task's cooperative `ctx.Done()` checks unwind it.
- **Progress** updates are ignored once a job is in a terminal state, so late channel
  writes can't resurrect a finished job's progress bar.
- **Cleanup**: `ClearFinishedJobs` removes all terminal jobs (and their log files);
  `CleanupOldJobs(maxAge)` removes terminal jobs older than a cutoff (the
  cleanup endpoint uses 24h).

### Log coloring

`log_colors.go` (`ParseAndColorLogContent`) wraps each log line in a CSS class based
on its level (`ERROR`→red, `WARN`→yellow) and on `color=...` hints embedded in
`INFO` lines (`green`, `blue`, `orange`, `violet`, `cyan`, …). The features emit
those `color=` hints in their log calls so the job log reads like a colored console.
The logs endpoint can return raw text, the colored HTML fragment (for HTMX
polling), or a full-screen HTML page.

### Webhooks

When `jobs.webhooks.enabled` is set and the finished job's type matches the
configured list (or `*`), `executeWebhook` renders the configured command template
with `{{.Name}}`, `{{.Type}}`, `{{.Status}}`, `{{.Message}}`, and `{{.Duration}}`,
then runs it via `/bin/sh -c` in its own process group with a 30-second kill timer.
This is how job completions can be pushed to notification services.

## Endpoints

Registered in `src/features/jobs/routes.go`. Several endpoints return JSON with
HATEOAS-style `_links` (self / progress / logs).

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/jobs` | Render the Jobs section | Section |
| GET | `/jobs/active` | Active (running/pending) jobs, newest first | Partial |
| GET | `/jobs/list` | All jobs, optionally `?prefix=` filtered by type | Partial |
| GET | `/jobs/latest` | The 5 most recent jobs | Partial |
| GET | `/jobs/count` | Count of `?filter=active\|all` jobs, formatted `(N)` | Text |
| GET | `/jobs/all` | All jobs as JSON (with `_links`) | JSON |
| POST | `/jobs/start/:type` | Start a job of the given type (`?name=`) | Toast (job) |
| POST | `/jobs/clear-finished` | Remove all terminal jobs + their logs | Toast |
| GET | `/jobs/:id` | Single job status as JSON (with `_links`) | JSON |
| GET | `/jobs/:id/progress` | Progress-bar partial (`HX-Trigger: done` when terminal) | Partial |
| GET | `/jobs/:id/logs` | Job logs — `?color=true` for HTML, else raw text | Text / HTML |
| POST | `/jobs/:id/cancel` | Cancel a job, returns refreshed job card | Partial |

## Configuration

The jobs feature reads a `jobs` block from the global config:

```yaml
jobs:
  log: true                 # write per-job log files
  log_path: /app/logs/jobs  # directory for the dated log files
  webhooks:
    enabled: false          # fire a command on job completion
    job_types: ["*"]        # which job types trigger the webhook ("*" = all)
    command: 'curl -X POST ... -d "{{.Name}} {{.Status}}"'  # templated shell command
```

When `log` is false, jobs use a discard logger and the logs endpoint reports no logs.

## Related

- [Importing](./importing.md), [Lyrics](./lyrics.md), [Reorganize](./reorganize.md), [Metadata](./metadata.md), [Metrics](./metrics.md) — all run their heavy work as jobs.
- [Configuration](./config.md) — the `jobs` config block and webhook settings.
- [Jobs (overview)](../jobs.md) — original high-level notes on the job system.
- [API reference](../api.md) — response-type semantics.
