# 🎧 Soulsolid

A music library management application with Telegram bot integration.

## Features

- Music library management and organization
- Telegram bot for remote control and notifications
- Automatic music importing from directories
- Device synchronization (iPod, USB drives, etc.)
- Metadata tagging with multiple providers (AcoustID, Discogs, MusicBrainz)
- Web interface for library browsing and management
- Job queue system for background processing

## Quick Start

### 🦭 Container Usage

The application can run without copying `config.yaml` into the container. If no config file exists, it will automatically create one with sensible defaults. Use environment variables to override specific settings:

```bash
# Build the image
podman build -t soulsolid .

# Run with environment variables (config.yaml will be auto-created if missing)
podman run -d \
  --name soulsolid \
  -p 3535:3535 \
  -v /host/music:/app/library \
  -v /host/downloads:/app/downloads \
  -v /host/logs:/app/logs \
  -v /host/library.db:/app/library.db \
  -v /host/config.yaml:/app/config.yaml \
  -e TELEGRAM_TOKEN="your_token" \
  soulsolid
```

The web interface will be available at `http://localhost:3535`.

## ⚙️ Configuration

The `config.yaml` file contains all application settings. Key sections:

- **telegram**: Bot configuration and allowed users
- **library**: Path to your music library
- **import**: Import settings and file organization rules
- **sync**: Device synchronization configuration
- **tag**: Metadata provider settings (Not implemented yet)
- **server**: Web server configuration
- **downloaders**: Plugin downloader configurations

**Security Note**: Never commit sensitive values like tokens or API keys to version control. Use environment variables instead.


### Automatic Configuration

If no `config.yaml` file exists when the application starts, it will automatically create one with sensible default values. This means you can run Soulsolid out of the box without any configuration!

### Default Configuration Values

When auto-generated, the config file includes these defaults:

| Setting | Default Value | Description |
|---------|---------------|-------------|
| **Library Path** | `./music` | Where your music library is stored |
| **Download Path** | `./downloads` | Where downloaded music is saved |
| **Database Path** | `./library.db` | SQLite database location |
| **Jobs Log Path** | `./logs/jobs` | Where job logs are stored |
| **Server Port** | `3535` | Web interface port |
| **Logger Level** | `info` | Logging verbosity (debug, info, warn, error) |
| **Logger Format** | `text` | Log output format (text, json) |
| **Import Move** | `false` | Whether to move files during import (vs copy) |
| **Import Duplicates** | `queue` | How to handle duplicate files (replace, skip, queue) |
| **Jobs Log** | `true` | Enable job logging |
| **Artwork Embedded** | `true` | Embed album artwork in audio files |
| **Artwork Size** | `1000px` | Embedded artwork size |
| **Artwork Quality** | `85%` | JPEG quality for embedded artwork |

### Environment Variables

#### Available Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TELEGRAM_TOKEN` | Telegram bot token | - |


For security, sensitive values should be set via environment variables rather than stored in config.yaml:

```bash
# Required for Telegram bot functionality
export TELEGRAM_TOKEN="your_telegram_bot_token_here"
```

### Development

If you are lucky enough to have nix. You can simple run the following command the all dev dependencies will be setup for you.
```bash
nix-shell dev.nix
```
### Building CSS

### Logs
Application logs are written to stdout/stderr. For more detailed logging, check the job logs in `./logs/jobs/` by default.

## License

[Add your license information here]

---
