---
weight: 50
title: "Downloading"
description: "Downloading music through plugins."
icon: "article"
date: "2025-09-09T21:10:28+02:00"
lastmod: "2025-09-09T21:10:28+02:00"
draft: false
toc: true
---

Soulsolid supports downloading music through pluggable downloaders. The system is agnostic to specific music services: each service is supported by a plugin, and developers can create plugins for new platforms (see the [Plugin Development Guide](plugins.md)).

Downloaded tracks are tagged with metadata and land in the configured `downloadPath`, from where they can be [imported](importing.md) into the library.

## Installing Plugins

Downloaders are Go plugins loaded at runtime. There are two ways to install one: build it into your container image, or let Soulsolid build it from a Git repository on startup.

### Building a Plugin into the Image

You can extend the Soulsolid image to include a plugin. This is the repository's [`Containerfile.plugin`](https://github.com/contre95/soulsolid/blob/main/Containerfile.plugin):

```dockerfile
FROM contre95/soulsolid:beta
RUN git clone https://github.com/contre95/soulsolid-dummy-plugin /tmp/plugin
WORKDIR /tmp/plugin
RUN go mod edit -replace=github.com/contre95/soulsolid=/app
RUN go mod tidy
RUN go build -buildmode=plugin -o /app/plugins/dummy/plugin.so .
WORKDIR /app
```

This clones the dummy plugin repository, replaces the module path to point to the Soulsolid source in the image, tidies the modules, builds the plugin as a shared object, and places it in the plugins directory. Point the plugin's `path` at the resulting `.so` file.

### Building from a Git Repository

Soulsolid can also build plugins directly from Git repositories. Specify a `url` instead of a `path`, and the application will clone, build, and load the plugin on startup. This requires `git` and `go` in `PATH` (both are included in the official image). See [Building from Git Repositories](plugins.md#building-from-git-repositories) for how it works and its limitations.

## Configuration

Plugins are configured under the `downloaders` section:

```yaml
downloaders:
  plugins:
    - name: dummy
      url: https://github.com/contre95/soulsolid-dummy-plugin # build from git...
      # path: /app/plugins/dummy/plugin.so                   # ...or load a pre-built .so
      icon: https://soulsolid-demo.contre.io/img/galaxy.png
      config: {} # plugin specific settings, e.g. api keys
  artwork:
    embedded:
      enabled: true
      size: 1000   # max dimension in pixels
      quality: 85  # JPEG quality (0-100)
```

For a complete configuration example, see the [Configuration](config.md) documentation.

## Downloading Process

The download process varies by plugin implementation, but generally follows this pattern:

1. **Authentication**: The plugin handles authentication with the music service
2. **Metadata Retrieval**: Fetches track/album metadata from the service's API
3. **Audio Download**: Downloads audio data in the requested format
4. **Format Processing**: Handles any decryption or format conversion if needed
5. **File Writing**: Saves the audio file to the configured download directory

## Tagging Process

After downloading, Soulsolid embeds comprehensive metadata into the audio files:

### For MP3 files (ID3v2 tags):

- Title, Artist, Album, Year, Genre
- Track/Position in set, Disc number
- ISRC, BPM, ReplayGain
- Album art (downloaded from the service)
- Publisher, Barcode, Composer, Lyrics

### For FLAC files (Vorbis comments):

- Similar metadata fields as MP3
- Additional Vorbis-specific fields like VERSION, DISCNUMBER
