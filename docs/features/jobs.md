---
weight: 100
title: "Jobs"
description: "Background jobs: progress, logs, cancelling, and webhooks."
icon: "manufacturing"
draft: false
toc: true
---

Anything in Soulsolid that can take a while (importing a folder, downloading an album,
running a lyrics analysis, reorganizing files, refreshing metrics) runs as a background
**job**. You can keep using the app while it runs, follow its progress, read its log,
and cancel it. Soulsolid can also run a **webhook** command when a job finishes, for
example to send you a notification or tell your media server to rescan.

## What you can do

- **Follow** every job's progress and status in the **All Jobs** section.
- **Read the log** of any job, live while it runs.
- **Cancel** a running or waiting job.
- **Clear** finished jobs from the list.
- Run a **webhook** command when certain jobs finish.

## How jobs run

**Only one job runs at a time.** If you start a job while another is running, it waits
in line and starts automatically when the current one finishes, oldest first. This keeps
jobs from getting in each other's way, for example an import reading files that a
download is still writing.

Each job has a status:

| Status | Meaning |
|--------|---------|
| **Pending** | Waiting for the running job to finish. |
| **Running** | In progress, with a progress bar and a message saying what it's doing. |
| **Completed** | Finished. Some jobs (like imports) can complete *with errors* when only some items failed. |
| **Failed** | Stopped because of an error. |
| **Cancelled** | You cancelled it. |

Jobs are kept in memory, so the jobs list is **emptied when Soulsolid restarts**. Their
log files stay on disk.

## Job logs

When job logging is enabled (`jobs.log: true`), every job writes its own log file in
`jobs.log_path`, named after the date and the job. You can read it from the job's card
in the app:

- inline, refreshed live while the job runs,
- **full screen**, with colors,
- **raw**, as plain text.

Logs are color-coded so errors (red) and warnings (yellow) stand out. When a job fails
on a specific track, the log line includes a link to open that track in the
[tag editor](./metadata.md) and fix it by hand. See [Analyze → Job Logs](../analyze.md#job-logs)
for an example.

## Cancelling and clearing

- **Cancel** stops a running job as soon as it safely can (usually after the current
  track), or removes a pending one from the line.
- **Clear Finished Jobs** removes every completed, failed and cancelled job from the
  list, and deletes their log files.

## Webhooks

A webhook is a shell command Soulsolid runs when a job finishes. Use it to send a
notification (ntfy, Telegram, Discord…), trigger a media server rescan, start a backup,
or anything else you can do from a shell. See [Jobs configuration](../jobs.md) for how to
set it up and ready-to-use examples.

Webhooks are configured with:

- `job_trigger`: which [job types](../jobs.md#job-types) trigger the command (`"*"` for all).
- `command`: the command to run. It can use these placeholders:

| Placeholder | Value |
|-------------|-------|
| `{{.Name}}` | The job's name, e.g. `Download Album` or `Download Playlist: Road trip` |
| `{{.Type}}` | The [job type](../jobs.md#job-types), e.g. `download_album` |
| `{{.Status}}` | `completed`, `failed` or `cancelled` |
| `{{.Message}}` | The job's final message |
| `{{.Duration}}` | How long the job took |

You can also use simple conditions, such as
`{{if eq .Status "failed"}}...{{end}}` or `{{or .Message "no message"}}`.

### How webhook commands run

Each webhook shows up in the jobs list as its own job, named `Webhook: <job name>`, so
you can check that it worked:

- The command runs with `/bin/sh`. What it prints is saved in the webhook job's log
  (when job logging is enabled).
- If the command exits with an error, or takes longer than **30 seconds**, the webhook
  job fails. **Cancel** stops the command and everything it started.
- Nothing the command starts keeps running after it finishes, including background
  processes (`curl ... &`).
- Webhooks **don't wait in line**: they start right away, even while another job runs,
  so notifications aren't delayed.
- Webhook jobs never trigger webhooks themselves, even with `job_trigger: ["*"]`.

### Passing secrets to webhooks

For safety, a webhook command does **not** see Soulsolid's environment variables, so API
keys you've given Soulsolid can't leak into it. It only sees:

- `PATH`, so commands like `curl` can be found.
- Variables whose name starts with **`SOULSOLID_WEBHOOK_`**, passed through as they are.
- The job's values as `SOULSOLID_JOB_NAME`, `SOULSOLID_JOB_TYPE`,
  `SOULSOLID_JOB_STATUS`, `SOULSOLID_JOB_MESSAGE` and `SOULSOLID_JOB_DURATION`.

Put tokens and URLs in `SOULSOLID_WEBHOOK_*` variables so they never appear in
`config.yaml` or in logs:

```yaml
command: curl -s -d "{{.Name}} {{.Status}}" "https://ntfy.sh/$SOULSOLID_WEBHOOK_NTFY_TOPIC"
```

Anything else the command needs, such as `HOME` or `SSL_CERT_FILE`, has to be passed
the same way and exported in the command, e.g.
`export SSL_CERT_FILE="$SOULSOLID_WEBHOOK_SSL_CERT_FILE"; curl ...`.

### Track titles and names are safe

Job names and messages can contain text that comes from your files, like track titles
or playlist names. Soulsolid makes sure that text is always treated as plain text: each
placeholder becomes exactly **one argument** of the command, however it's quoted. A
track titled `"; rm -rf ~; "` is just a title, not a command.

There's one thing to watch out for: if your command hands a value to **another shell**
(`sh -c`, `bash -c`, `ssh host '...'`, `su -c`, `eval`), that shell *will* run it as
code. Pass the value as an argument instead, or read the variable inside:

```yaml
command: sh -c 'notify "$1"' _ {{.Name}}            # value as an argument
command: sh -c 'notify "$SOULSOLID_JOB_NAME"'       # the inner shell reads the variable
```

To keep things safe, a webhook **doesn't run** if its command:

- uses a placeholder inside `$(...)`, backticks, heredocs or `$'...'`, or anywhere after
  one of these (use the `SOULSOLID_JOB_*` variables there instead);
- formats a value with `printf`, `urlquery`, `html`, `js` or `slice` (do that formatting
  in the command itself, for example with `jq`).

### Sending JSON

Placeholders are inserted as they are, so a `"` or `\` in a track title can break a JSON
body written by hand, such as `-d '{"text": "{{.Name}}"}'`. That can't run anything,
but the service you're calling may reject the request. To always send valid JSON, build
it with [`jq`](https://jqlang.org) (it must be installed where Soulsolid runs) and pipe
it to `curl`:

```yaml
command: >-
  jq -n --arg name "{{.Name}}" --arg status "{{.Status}}"
  '{text: "\($name) \($status)"}'
  | curl -X POST -H 'Content-Type: application/json' -d @- https://example.com/hook
```

## Configuration

```yaml
jobs:
  log: true                 # write a log file for every job
  log_path: /app/logs/jobs  # where log files are written
  webhooks:
    enabled: false          # run a command when a job finishes
    job_trigger: ["*"]      # which job types trigger it ("*" = all)
    command: 'curl -d "{{.Name}} {{.Status}}" ntfy.sh/mytopic'
```

When `log` is off, jobs still run but no logs are kept.

## Related

- [Jobs configuration](../jobs.md): job types and webhook examples.
- [Importing](./importing.md), [Downloading](./downloading.md), [Lyrics](./lyrics.md), [Reorganize](./reorganize.md), [Merge](./merge.md), [Metadata](./metadata.md), [Metrics](./metrics.md): everything that runs as a job.
