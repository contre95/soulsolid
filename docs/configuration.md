# Configuration Guide

Soulsolid supports configuration through YAML files and environment variables. This guide covers both methods.

## Configuration Methods

1. **YAML File**: Primary configuration method via `config.yaml`
2. **Environment Variables**: Override specific settings with `SS_` prefix
3. **Legacy Environment Variables**: Support for older variable names (backward compatibility)

## YAML Configuration

The main configuration file is `config.yaml`. A sample configuration file (`config.example.yaml`) is provided with default values.

### Structure Overview

```yaml
# Core paths
libraryPath: ./music_library
downloadPath: ./downloads

# Telegram integration
telegram:
  enabled: false
  token: <token>
  allowedUsers:
    - contre95
  bot_handle: MusicarrDemoBot

# Logging configuration
logger:
  enabled: true
  level: info
  format: text
  htmx_debug: false

# Downloaders and plugins
downloaders:
  plugins:
    - name: dummy
      path: ../soulsolid-dummy-plugin/plugin.so
      icon: https://demo2.contre.io/img/galaxy.png
      config: {} # Plugin specific config.
  artwork:
    embedded:
      enabled: true
      size: 1000
      quality: 85

# Server settings
server:
  show_routes: false
  port: 3535

# Database
database:
  path: ./library.db

# Import settings
import:
  move: false
  always_queue: false
  duplicates: skip
  auto_start_watcher: false
  allow_missing_metadata: true
  paths:
    compilations: "%artistfolder{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}"
    album:soundtrack: "%artistfolder{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}"
    album:single: "%artistfolder{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}"
    album:ep: "%artistfolder{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}"
    default_path: "%artistfolder{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}"

# Metadata providers
metadata:
  providers:
    acoustid:
      enabled: true
      secret: <acoustid_secret>
    deezer:
      enabled: true
    discogs:
      enabled: true
      secret: <discogs_token>
    musicbrainz:
      enabled: true

# Lyrics providers
lyrics:
  providers:
    lrclib:
      enabled: true

# Device synchronization
sync:
  enabled: true
  devices:
    - uuid: 8722-166E
      name: iPod
      sync_path: SoulMusic

# Job management
jobs:
  log: true
  log_path: ./logs/jobs
  webhooks:
    enabled: true
    job_types:
      - directory_import
      - download_album
      - dap_sync
    command: "TEXT=\"🎵 Job {{.Name}} ({{.Type}}) {{.Status}}\\n📝 {{.Message}}\\n⏱️ Duration: {{.Duration}}\"\ncurl -X POST -H 'Content-Type: application/json' \\\n  -d '{\"chat_id\": \"<chat_id>\", \"text\": \"'\"$TEXT\"'\", \"parse_mode\": \"HTML\"}' \\\n  https://api.telegram.org/bot<bot_token>/sendMessage\n"
```

## Environment Variables

All configuration options can be overridden using environment variables with the `SS_` prefix. Environment variables take precedence over YAML configuration.

### Naming Convention

YAML paths are converted to environment variables by:
1. Adding `SS_` prefix
2. Replacing dots (`.`) and colons (`:`) with underscores (`_`)
3. Converting to UPPERCASE

**Examples**:
- `logger.enabled` → `SS_LOGGER_ENABLED`
- `import.paths.album:soundtrack` → `SS_IMPORT_PATHS_ALBUM_SOUNDTRACK`
- `lyrics.providers.lrclib.prefer_synced` → `SS_LYRICS_PROVIDERS_LRCLIB_PREFER_SYNCED`

### Basic Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `libraryPath` | `SS_LIBRARYPATH` | string | `./music_library` | Path to music library |
| `downloadPath` | `SS_DOWNLOADPATH` | string | `./downloads` | Path for downloads |
| `database.path` | `SS_DATABASE_PATH` | string | `./library.db` | SQLite database path |
| `server.port` | `SS_SERVER_PORT` | integer | `3535` | Web server port |
| `server.show_routes` | `SS_SERVER_SHOW_ROUTES` | boolean | `false` | Show Fiber routes |

### Telegram Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `telegram.enabled` | `SS_TELEGRAM_ENABLED` | boolean | `false` | Enable Telegram bot |
| `telegram.token` | `SS_TELEGRAM_TOKEN` | string | `""` | Telegram bot token |
| `telegram.bot_handle` | `SS_TELEGRAM_BOT_HANDLE` | string | `MusicarrDemoBot` | Bot username (without @) |
| `telegram.allowedUsers` | `SS_TELEGRAM_ALLOWEDUSERS` | array | `[]` | Allowed users (see Array section) |

### Logging Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `logger.enabled` | `SS_LOGGER_ENABLED` | boolean | `true` | Enable logging |
| `logger.level` | `SS_LOGGER_LEVEL` | string | `info` | Log level (debug, info, warn, error) |
| `logger.format` | `SS_LOGGER_FORMAT` | string | `text` | Log format (text, json) |
| `logger.htmx_debug` | `SS_LOGGER_HTMX_DEBUG` | boolean | `false` | Enable HTMX debugging |

### Import Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `import.move` | `SS_IMPORT_MOVE` | boolean | `false` | Move files instead of copying |
| `import.always_queue` | `SS_IMPORT_ALWAYS_QUEUE` | boolean | `false` | Always queue imports for review |
| `import.duplicates` | `SS_IMPORT_DUPLICATES` | string | `skip` | Duplicate handling (skip, queue, replace) |
| `import.auto_start_watcher` | `SS_IMPORT_AUTO_START_WATCHER` | boolean | `false` | Auto-start directory watcher |
| `import.allow_missing_metadata` | `SS_IMPORT_ALLOW_MISSING_METADATA` | boolean | `true` | Allow tracks without artist/album metadata |

### Path Templates

| YAML Path | Environment Variable | Type | Description |
|-----------|----------------------|------|-------------|
| `import.paths.compilations` | `SS_IMPORT_PATHS_COMPILATIONS` | string | Path template for compilations |
| `import.paths.album:soundtrack` | `SS_IMPORT_PATHS_ALBUM_SOUNDTRACK` | string | Path template for soundtracks |
| `import.paths.album:single` | `SS_IMPORT_PATHS_ALBUM_SINGLE` | string | Path template for singles |
| `import.paths.album:ep` | `SS_IMPORT_PATHS_ALBUM_EP` | string | Path template for EPs |
| `import.paths.default_path` | `SS_IMPORT_PATHS_DEFAULT_PATH` | string | Default path template |

### Metadata Providers

Metadata providers are configured as a map. The following providers are supported:

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `metadata.providers.acoustid.enabled` | `SS_METADATA_PROVIDERS_ACOUSTID_ENABLED` | boolean | `true` | Enable AcoustID |
| `metadata.providers.acoustid.secret` | `SS_METADATA_PROVIDERS_ACOUSTID_SECRET` | string | `""` | AcoustID API secret |
| `metadata.providers.deezer.enabled` | `SS_METADATA_PROVIDERS_DEEZER_ENABLED` | boolean | `true` | Enable Deezer |
| `metadata.providers.discogs.enabled` | `SS_METADATA_PROVIDERS_DISCOGS_ENABLED` | boolean | `true` | Enable Discogs |
| `metadata.providers.discogs.secret` | `SS_METADATA_PROVIDERS_DISCOGS_SECRET` | string | `""` | Discogs API token |
| `metadata.providers.musicbrainz.enabled` | `SS_METADATA_PROVIDERS_MUSICBRAINZ_ENABLED` | boolean | `true` | Enable MusicBrainz |

**Note**: For any provider `[name]`, use `SS_METADATA_PROVIDERS_[NAME]_ENABLED` and `SS_METADATA_PROVIDERS_[NAME]_SECRET`.

### Lyrics Providers

Lyrics providers are configured as a map. Replace `[provider]` with the provider name (e.g., `lrclib`, `genius`, `tekstowo`).

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `lyrics.providers.[provider].enabled` | `SS_LYRICS_PROVIDERS_[PROVIDER]_ENABLED` | boolean | `true` | Enable lyrics provider |
| `lyrics.providers.[provider].prefer_synced` | `SS_LYRICS_PROVIDERS_[PROVIDER]_PREFER_SYNCED` | boolean | `false` | Prefer synced lyrics |

**Examples**:
- `lyrics.providers.lrclib.enabled` → `SS_LYRICS_PROVIDERS_LRCLIB_ENABLED`
- `lyrics.providers.lrclib.prefer_synced` → `SS_LYRICS_PROVIDERS_LRCLIB_PREFER_SYNCED`

### Downloader Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `downloaders.artwork.embedded.enabled` | `SS_DOWNLOADERS_ARTWORK_EMBEDDED_ENABLED` | boolean | `true` | Enable embedded artwork |
| `downloaders.artwork.embedded.size` | `SS_DOWNLOADERS_ARTWORK_EMBEDDED_SIZE` | integer | `1000` | Artwork size (pixels) |
| `downloaders.artwork.embedded.quality` | `SS_DOWNLOADERS_ARTWORK_EMBEDDED_QUALITY` | integer | `85` | Artwork quality (1-100) |

**Note**: `downloaders.plugins` should be configured only via YAML, not environment variables.

### Sync Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `sync.enabled` | `SS_SYNC_ENABLED` | boolean | `true` | Enable device sync |
| `sync.devices` | `SS_SYNC_DEVICES_*` | array | `[]` | Sync devices (see Array section) |

### Job Configuration

| YAML Path | Environment Variable | Type | Default | Description |
|-----------|----------------------|------|---------|-------------|
| `jobs.log` | `SS_JOBS_LOG` | boolean | `true` | Enable job logging |
| `jobs.log_path` | `SS_JOBS_LOG_PATH` | string | `./logs/jobs` | Job log directory |
| `jobs.webhooks.enabled` | `SS_JOBS_WEBHOOKS_ENABLED` | boolean | `true` | Enable webhooks |
| `jobs.webhooks.job_types` | `SS_JOBS_WEBHOOKS_JOB_TYPES` | array | `[]` | Job types for webhooks |
| `jobs.webhooks.command` | `SS_JOBS_WEBHOOKS_COMMAND` | string | `""` | Webhook command |

## Array Configuration

### String Arrays (e.g., `telegram.allowedUsers`)

**Method 1: Comma-separated string**
```bash
export SS_TELEGRAM_ALLOWEDUSERS="user1,user2,user3"
```

**Method 2: Indexed environment variables**
```bash
export SS_TELEGRAM_ALLOWEDUSERS_0="user1"
export SS_TELEGRAM_ALLOWEDUSERS_1="user2"
export SS_TELEGRAM_ALLOWEDUSERS_2="user3"
```

### Object Arrays (e.g., `sync.devices`)

Use indexed environment variables for each field:

```bash
# Device 0
export SS_SYNC_DEVICES_0_UUID="8722-166E"
export SS_SYNC_DEVICES_0_NAME="iPod"
export SS_SYNC_DEVICES_0_SYNC_PATH="SoulMusic"

# Device 1
export SS_SYNC_DEVICES_1_UUID="ABCD-1234"
export SS_SYNC_DEVICES_1_NAME="Walkman"
export SS_SYNC_DEVICES_1_SYNC_PATH="PortableMusic"
```

### Job Type Arrays (e.g., `jobs.webhooks.job_types`)

Similar to string arrays, use comma-separated or indexed format:
```bash
# Comma-separated
export SS_JOBS_WEBHOOKS_JOB_TYPES="directory_import,download_album,dap_sync"

# Indexed
export SS_JOBS_WEBHOOKS_JOB_TYPES_0="directory_import"
export SS_JOBS_WEBHOOKS_JOB_TYPES_1="download_album"
export SS_JOBS_WEBHOOKS_JOB_TYPES_2="dap_sync"
```

## Legacy Environment Variables

For backward compatibility, these legacy environment variables are still supported:

| Legacy Variable | New Equivalent | Description |
|-----------------|----------------|-------------|
| `TELEGRAM_TOKEN` | `SS_TELEGRAM_TOKEN` | Telegram bot token |
| `DISCOGS_API_KEY` | `SS_METADATA_PROVIDERS_DISCOGS_SECRET` | Discogs API token |
| `ACOUSTID_CLIENT_KEY` | `SS_METADATA_PROVIDERS_ACOUSTID_SECRET` | AcoustID API secret |

## Usage Examples

### Docker/Podman
```bash
docker run -d \
  -p 3535:3535 \
  -v /host/music:/app/library \
  -v /host/downloads:/app/downloads \
  -e SS_LOGGER_ENABLED=false \
  -e SS_TELEGRAM_ENABLED=true \
  -e SS_TELEGRAM_TOKEN="your_token" \
  -e SS_TELEGRAM_ALLOWEDUSERS="user1,user2" \
  -e SS_SYNC_ENABLED=true \
  -e SS_SYNC_DEVICES_0_UUID="1234-5678" \
  -e SS_SYNC_DEVICES_0_NAME="MyDevice" \
  soulsolid
```

### Systemd Service
```ini
[Service]
Environment=SS_LOGGER_ENABLED=true
Environment=SS_LOGGER_LEVEL=info
Environment=SS_SERVER_PORT=3535
Environment=SS_TELEGRAM_TOKEN=your_token
Environment=SS_TELEGRAM_ALLOWEDUSERS=admin
```

### Development
```bash
# Create .env file
cp .env.example .env
# Edit .env with your values
source .env
go run src/main.go
```

## Configuration Priority

1. **Environment Variables** (highest priority)
2. **YAML Configuration File** (`config.yaml`)
3. **Default Values** (lowest priority)

When both YAML and environment variables are set, environment variables take precedence.

## Validation

Configuration is validated using `go-playground/validator`. Required fields must be present:
- `libraryPath`: Path to music library
- `downloadPath`: Path for downloads
- `database.path`: SQLite database path

If validation fails, the application will exit with an error message.

## Automatic Configuration Creation

If `config.yaml` doesn't exist when the application starts:
1. A default configuration is created
2. The file is saved to `config.yaml`
3. Required directories are created (`libraryPath`, `downloadPath`)
4. Application starts with default values

## Web UI Configuration

Configuration can be modified via the web UI at `/settings`. Changes are saved to `config.yaml`.

**Note**: Environment variable overrides will still apply even if changed via the web UI. To permanently change a value that's set via environment variable, either:
1. Remove the environment variable
2. Update the environment variable value
3. Or modify the value in both the environment variable and web UI
