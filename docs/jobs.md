---
weight: 70
title: "Jobs"
description: "Background jobs, job logs and webhooks."
icon: "article"
date: "2025-09-09T21:09:51+02:00"
lastmod: "2025-09-09T21:09:51+02:00"
draft: false
toc: true
---

The `jobs` section configures how Soulsolid handles background tasks. These jobs are stored in runtime and their logs are persisted in files under a specified location. The default location for these logs is `./logs/jobs`.

Here's an example configuration:

```yaml
jobs:
  log: true
  log_path: ./logs/jobs
  webhooks:
    enabled: true
    job_trigger:
      - directory_import
      - download_album
    command: | # ntfy
      curl -d "Backup successful 😀" ntfy.sh/mytopic
```

- **log**: Enable or disable job logging.
- **log_path**: The directory where job logs are stored.
- **webhooks**: Configuration for sending notifications about job status.
  - **enabled**: Enable or disable webhooks.
  - **job_trigger**: List of job types to send notifications for (`"*"` = all). Formerly `job_types`, which is still read but deprecated.
  - **command**: Shell command executed via `/bin/sh -c` when a matching job finishes. Runs asynchronously with a 30-second timeout. Supports Go template syntax with these variables: `{{.Name}}`, `{{.Type}}`, `{{.Status}}`, `{{.Message}}`, `{{.Duration}}`.

## Job Scheduling

Only one job runs at a time. Additional jobs are queued and start automatically when the current job finishes.

## Job Types

| Type | Triggered by |
|------|-------------|
| `directory_import` | Importing a directory of audio files |
| `calculate_metrics` | Recalculating the library metrics shown on the dashboard |
| `download_track` | Downloading a single track via a plugin |
| `download_album` | Downloading a full album via a plugin |
| `download_artist` | Downloading all albums for an artist via a plugin |
| `download_tracks` | Downloading multiple tracks in bulk |
| `download_playlist` | Downloading a playlist via a plugin |
| `analyze_acoustid` | Running AcoustID fingerprint analysis on the library |
| `analyze_lyrics` | Running lyrics analysis on the library |
| `analyze_reorganize` | Reorganizing library file paths |
| `analyze_merge` | Merging artists, albums, genres or tracks |

## Notification Examples

### Telegram

Sends a Telegram message with job details:

```yaml
jobs:
  webhooks:
    enabled: true
    job_trigger:
      - directory_import
      - download_album
    command: |
      TEXT="🎵 Job {{.Name}} ({{.Type}}) {{.Status}}\n📝 {{.Message}}\n⏱️ Duration: {{.Duration}}" && \
        curl -X POST -H 'Content-Type: application/json' \
          -d "{\"chat_id\": \"<chat_id>\", \"text\": \"$TEXT\", \"parse_mode\": \"HTML\"}" \
          https://api.telegram.org/bot<bot_token>/sendMessage
```

### ntfy

Sends a notification to an ntfy.sh topic for every job:

```yaml
jobs:
  webhooks:
    enabled: true
    job_trigger:
      - "*"
    command: curl -d "Job {{.Name}} {{.Status}} 😀" ntfy.sh/mytopic
```

### Emby

Triggers a library rescan in Emby after imports:

```yaml
jobs:
  webhooks:
    enabled: true
    job_trigger:
      - directory_import
    command: curl -X POST 'http://localhost:8096/Emby/Library/Refresh?api_key=your_emby_api_key'
```
