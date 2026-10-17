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
and runs it as a **`webhook` job** named `Webhook: <job name>`. This is how job
completions can be pushed to notification services.

Webhook jobs appear in the jobs list like any other job:

- The command runs via `/bin/sh -c` in its own process group. Its **output** goes
  into the job's log as it runs: stdout as `INFO` lines and stderr as `WARN` lines,
  up to 64 KiB per stream. Output is only kept when `jobs.log` is enabled.
- A non-zero exit fails the job, with the exit status as its message. So does
  running past the **30-second timeout**. **Cancel** kills the command and anything
  it started.
- Nothing the command starts outlives the job. Background processes (`curl ... &`) are
  stopped when the shell exits. If one still held the output open, the job completes
  with a warning in its log, and output written after the shell exited is dropped.
- Webhook jobs **skip the queue**. They start immediately, even while another job
  runs, and never hold up queued jobs, so notifications aren't delayed.
- Webhook jobs never trigger webhooks themselves, even with `job_trigger: ["*"]`.
- The triggering job's log records `Webhook job started` with the webhook job's ID.

#### Environment

A webhook command doesn't inherit the server's environment, so secrets passed to
Soulsolid (for example through `!env_var`) can't leak into it. It only sees:

- `PATH`, so commands like `curl` can be found.
- Server variables starting with **`SOULSOLID_WEBHOOK_`**, passed through unchanged.
- The job's `SOULSOLID_JOB_*` values (see below). They can't be overridden from
  the server environment.

Use `SOULSOLID_WEBHOOK_*` variables for tokens and URLs. That way they never appear
in `config.yaml`, in job metadata or in the logged command:

```yaml
command: curl -s -d "{{.Name}} {{.Status}}" "https://ntfy.sh/$SOULSOLID_WEBHOOK_NTFY_TOPIC"
```

Anything else the command needs, such as `HOME` or `SSL_CERT_FILE`, has to be passed
under that prefix and exported in the command, e.g.
`export SSL_CERT_FILE="$SOULSOLID_WEBHOOK_SSL_CERT_FILE"; curl ...`.

Job values never become part of the shell source. Job names and messages can contain
text the user or a file's tags control, such as playlist names and track titles. The
values are passed to the command as environment variables (`SOULSOLID_JOB_NAME`,
`SOULSOLID_JOB_TYPE`, `SOULSOLID_JOB_STATUS`, `SOULSOLID_JOB_MESSAGE`,
`SOULSOLID_JOB_DURATION`), and each placeholder becomes a reference to one, quoted for
where it sits in the command:

| Template | Rendered as |
|----------|-------------|
| `{{.Name}}` (unquoted) | `"${SOULSOLID_JOB_NAME}"` |
| `"{{.Name}}"` (inside double quotes) | `${SOULSOLID_JOB_NAME}` |
| `'{{.Name}}'` (inside single quotes) | `'"${SOULSOLID_JOB_NAME}"'` |

Either way a placeholder expands to exactly **one literal argument** of the webhook's
shell. A value like `"; rm -rf ~; "` isn't run by that shell, `*` isn't expanded into
file names, and a message like `x --data-binary @/etc/passwd` can't add options to
`curl`. Scripts can also read the `SOULSOLID_JOB_*` variables directly.

This protects values only from the webhook's own shell. A command that hands a value
to **another parser of shell code**, such as `sh -c`, `bash -c`, `ssh host '...'`,
`su -c` or `eval`, runs it as code there: with `sh -c 'echo {{.Name}}'`, a track titled
`x; rm -rf ~` runs `rm`. Pass the value as an argument instead, or let the inner shell
read the variable itself:

```yaml
command: sh -c 'notify "$1"' _ {{.Name}}            # value as a positional argument
command: sh -c 'notify "$SOULSOLID_JOB_NAME"'       # the inner shell reads the variable
```

Template logic uses the real values, so `{{if eq .Status "failed"}}...{{end}}`,
`{{or .Message "no message"}}` and `{{len .Name}}` work as expected. Functions that
would turn a job value into new text, such as `printf`, `urlquery`, `html`, `js` or
`slice`, are rejected and the webhook doesn't run. Do that formatting in the command
instead, for example with `jq` (see below).

Placeholders inside `$(...)`, backticks, heredocs or `$'...'`, or anywhere after one of
these in the command, are rejected and the webhook doesn't run, because their quoting
can't be tracked reliably. Use the environment variables there instead.

#### Building JSON payloads

Placeholders insert values verbatim, so a `"` or `\` in a playlist name or track title
will still break a hand-written JSON body such as
`-d '{"text": "{{.Name}}"}'`. That can't run a command, but the receiving service may
reject the request. To always send valid JSON, build the payload with a JSON
serializer such as [`jq`](https://jqlang.org), which must be installed where Soulsolid
runs, and pipe it to `curl`:

```yaml
command: >-
  jq -n --arg name "{{.Name}}" --arg status "{{.Status}}"
  '{text: "\($name) \($status)"}'
  | curl -X POST -H 'Content-Type: application/json' -d @- https://example.com/hook
```

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
    job_trigger: ["*"]      # which job types trigger the webhook ("*" = all)
    command: 'curl -X POST ... -d "{{.Name}} {{.Status}}"'  # templated shell command
```

When `log` is false, jobs use a discard logger and the logs endpoint reports no logs.

## Related

- [Importing](./importing.md), [Lyrics](./lyrics.md), [Reorganize](./reorganize.md), [Metadata](./metadata.md), [Metrics](./metrics.md) — all run their heavy work as jobs.
- [Configuration](./config.md) — the `jobs` config block and webhook settings.
- [Jobs (overview)](../jobs.md) — original high-level notes on the job system.
- [API reference](../api.md) — response-type semantics.
