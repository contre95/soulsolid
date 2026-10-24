---
weight: 20
title: "Configuration"
description: "The config.yaml reference."
icon: "article"
date: "2025-09-09T21:10:30+02:00"
lastmod: "2025-09-09T21:10:30+02:00"
draft: false
toc: true
---

The `config.yaml` file contains all application settings. Key sections:

- **libraryPath** / **downloadPath**: Where the library lives and where downloads land
- **import**: Import settings and file organization rules (see [Importing](importing.md) and [Paths](paths.md))
- **metadata**: Metadata provider settings (musicbrainz, deezer, discogs, acoustid)
- **lyrics**: Lyrics provider settings (see [Lyrics](lyrics.md))
- **downloaders**: External service configurations in the form of plugins. Supports both local `.so` files and automatic building from Git repositories (see [Downloading](downloading.md))
- **jobs**: Job logs and webhooks (see [Jobs](jobs.md))
- **server**, **database**, **logger**: Web server port, SQLite path and logging
- **telegram**: Bot configuration and allowed users (deprecated, will be removed in v0.30.0)

**Security Note**: Never commit sensitive values like tokens or API keys to version control. Use environment variables or secret files instead.

The config file is read from `/config/config.yaml` by default; set `SOULSOLID_CONFIG_PATH` to use another location.

### Automatic Configuration

If no `config.yaml` file exists when the application starts, it will automatically create one with sensible default values. This means you can run Soulsolid out of the box without any configuration!

### Example Configuration

This is the repository's [`config.example.yaml`](https://github.com/contre95/soulsolid/blob/main/config.example.yaml):

```yaml
libraryPath: ./music
downloadPath: ./downloads
# Deprecated: the Telegram bot will be removed in v0.30.0.
telegram:
  enabled: false
  # token: !env_var TELEGRAM_BOT_TOKEN
  allowedUsers:
    - <your telegram username>
  bot_handle: SoulsolidExampleBot # Without the @
logger:
  enabled: true
  level: info
  format: text
  htmx_debug: false
downloaders:
  plugins:
    - name: dummy
      url: https://github.com/contre95/soulsolid-dummy-plugin 
      # path: ../soulsolid-dummy-plugin/plugin.so # Optional if building the plugin
      icon: https://demo2.contre.io/img/galaxy.png
      config: {} # Plugin specific config. See https://git.sr.ht/~canuslector/soulsolid-deemix-plugin
  artwork:
    embedded:
      enabled: true
      size: 1000
      quality: 85
server:
  show_routes: false
  port: 3535
database:
  path: ./library.db # Path to the SQLite Database
import:
  move: false # If false tracks will be kepts in the folder where you are importing them from and copied from it to your 'libraryPath:'
  always_queue: false # When true, it will queue every single imported track for manual review.
  duplicates: queue # queue | skip | replace
  allow_missing_metadata: # Per-field: when true the missing value is filled with a fallback default on import, otherwise the track is sent to the manual review queue
    artist: false
    album: false
    title: false
    year: false
    genre: false
  paths:
    compilations: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:soundtrack: '%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:single: '%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:ep: '%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    default_path: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
  auto_start_watcher: false
metadata:
  providers:
    acoustid:
      enabled: true
      # secret: !env_var ACOUSTID_CLIENT_KEY # You can login with Musicbrainz -> https://acoustid.org/new-application
      # secret: !env_file ACOUSTID_CLIENT_KEY_FILE # Or load it from a file instead, e.g. a Docker secret
    deezer:
      enabled: true
    discogs:
      enabled: true
      # secret: !env_var DISCOGS_API_KEY # You can get it here -> https://www.discogs.com/settings/developers
      # secret: !env_file DISCOGS_API_KEY_FILE # Or load it from a file instead, e.g. a Docker secret
    musicbrainz:
      enabled: true
lyrics:
  providers:
    lrclib:
      enabled: true
      prefer_synced: false # Some programs like navidrome require synced lyrics
jobs:
  log: true
  log_path: ./logs/jobs
  webhooks:
    enabled: true
    job_trigger:
      - directory_import
      - download_album
    command: "echo hi"
```

### Environment Variables

Any config value can be read from an environment variable with the `!env_var` tag, or from a
file whose path is in an environment variable with the `!env_file` tag (the Docker/Kubernetes
secrets convention). The application fails to start if a referenced variable is not set, or
if the file can't be read or is empty.

```yaml
metadata:
  providers:
    discogs:
      secret: !env_var DISCOGS_API_KEY
    acoustid:
      secret: !env_file ACOUSTID_CLIENT_KEY_FILE # e.g. ACOUSTID_CLIENT_KEY_FILE=/run/secrets/acoustid_client_key
```

The variable names in the examples above are conventions, not fixed names. The app itself reads:

| Variable                | Description                                  | Default               |
| ----------------------- | -------------------------------------------- | --------------------- |
| `SOULSOLID_CONFIG_PATH` | Path to the config file                      | `/config/config.yaml` |
| `IMAGE_TAG`             | Application version shown in the UI          | `dev`                 |
