The `jobs` section configures how Soulsolid handles background tasks. These jobs are stored in runtime and their logs are persisted in files under a specified location. The default location for these logs is `./logs/jobs`.

Here's an example configuration:

```yaml
jobs:
  log: true
  log_path: ./logs/jobs
  webhooks:
    enabled: true
    job_types:
      - directory_import
      - download_album
      - dap_sync
    command: | # ntfy
      curl -d "Backup successful 😀" ntfy.sh/mytopic
```

- **log**: Enable or disable job logging.
- **log_path**: The directory where job logs are stored.
- **webhooks**: Configuration for sending notifications about job status.
  - **enabled**: Enable or disable webhooks.
  - **job_types**: List of job types to send notifications for.
  - **command**: The command to execute when a job status changes. This can be used to send notifications to various services.
    The following rendering names are allowed: Name, Type, Status, Message, Duration.

    **Notification Examples:**
    - **Telegram:** Sends a Telegram message with job details:
      ```bash
      TEXT="🎵 Job {{.Name}} ({{.Type}}) {{.Status}}\n📝 {{.Message}}\n⏱️ Duration: {{.Duration}}" && \
      curl -X POST -H 'Content-Type: application/json' \
      -d "{\"chat_id\": \"<chat_id>\", \"text\": \"$TEXT\", \"parse_mode\": \"HTML\"}" \
      https://api.telegram.org/bot<bot_token>/sendMessage
      ```
    - **ntfy:** Sends a notification to an ntfy.sh topic:
      ```bash
      curl -d "Backup finished 😀." ntfy.sh/mytopic
      ```
    - **Emby:** Triggers a library rescan in Emby:
      ```bash
      curl -X POST 'http://your_emby_server:8096/emby/Library/Media/Updated?api_key=your_emby_api_key'
      ```

