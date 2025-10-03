# 🎧 Soulsolid <img src="public/img/galaxy.png" width="50" alt="Galaxy">

A feature rich music organization app built for the music hoarder. Heavily under development, focused on ease of usage.

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
