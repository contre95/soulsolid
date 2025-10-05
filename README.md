<table>
  <tr>
    <td><img src="public/img/galaxy.png" width="50" alt="souldsolid"></td>
    <td><h1>Soulsolid</h1></td>
  </tr>
</table>
A feature rich music organization app built for the music hoarder. Heavily under development, focused on ease of usage.

## Features

- **Music Library Management**: Organize and browse albums, artists, and tracks
- **Downloading**: Download tracks and albums. 
- **Importing**: Import music from directories with automatic fingerprinting
- **Metadata Tagging**: Auto-tag using MusicBrainz and Discogs APIs
- **Sync with DAP**: Sync library with digital audio players
- **Telegram Integration**: Control via Telegram bot
- **Web UI**: Mobile-friendly interface for all operations 
- **Job Management**: Background processing for downloads, imports, and syncs

Documentation: https://soulsolid.contre.io

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

## Development

To set up the development environment:

### Option 1: Manual Setup

```bash
cp config.example.yaml config.yaml
npm run dev
go run ./src/main.go
```

### Option 2: Using Nix (recommended if you have Nix)

If you have Nix installed, use the provided dev.nix shell:

```bash
# Set up all dependencies (Node.js, Go, etc.) and run the necessary commands
nix-shell dev.nix
# Then, simply run:
go run ./src/main.go
```

The web interface will be available at `http://localhost:3535`.

