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

Soulsolid can download music from online music services through **downloader plugins**. Soulsolid itself isn't tied to any service: each service is supported by its own plugin, which you add to your configuration. Anyone can write a plugin for a new service (see the [Plugin Development Guide](plugins.md)).

This page covers installing and configuring plugins. For searching and downloading in the app, see [Downloading](features/downloading.md).

Downloaded tracks are tagged with metadata and land in the configured `downloadPath`, from where they can be [imported](importing.md) into the library.

## Installing Plugins

Plugins are loaded when Soulsolid starts. There are two ways to install one: let Soulsolid build it from its Git repository on startup (the easiest), or build it into your own container image.

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

This downloads the plugin's source, builds it against the Soulsolid version in the image and places it in `/app/plugins/dummy/plugin.so`. Point the plugin's `path` at that file. Building into the image means Soulsolid starts faster and doesn't need network access to load the plugin.

### Building from a Git Repository

Set the plugin's `url` to its Git repository instead of a `path`, and Soulsolid downloads, builds and loads it every time it starts. This needs `git` and `go` to be installed, which they are in the official image. Only the repository's default branch is used. See [Building from Git Repositories](plugins.md#building-from-git-repositories) for details.

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

The details depend on the plugin, but a download generally goes like this:

1. **Login**: the plugin logs in to the music service with the credentials in its `config`
2. **Track information**: it fetches the track or album details from the service
3. **Audio**: it downloads the audio
4. **Saving**: the file is saved to your `downloadPath`, tagged and with its cover art

Downloads run as background [jobs](features/jobs.md). Once done, [import](importing.md) them into your library, or turn on the download folder watcher to do it automatically.

## Tagging Process

After downloading, Soulsolid writes the track's details into the audio file's tags:

### For MP3 files (ID3v2 tags):

- Title, Artist, Album, Year, Genre
- Track/Position in set, Disc number
- ISRC, BPM, ReplayGain
- Album art (downloaded from the service)
- Publisher, Barcode, Composer, Lyrics

### For FLAC files (Vorbis comments):

- Similar metadata fields as MP3
- Additional Vorbis-specific fields like VERSION, DISCNUMBER
