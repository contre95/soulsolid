---
weight: 20
title: "Downloading"
description: "Searching external sources and downloading tracks, albums, artists, and playlists via plugins."
icon: "download"
draft: false
toc: true
---

The **Download** section lets you search external music services and download tracks,
albums, whole artists and playlists. Soulsolid doesn't come with any music service
built in: each service is added through a **downloader plugin**, so what you can search
and download depends on the plugins you have configured. See
[Downloading](../downloading.md) for how to install one.

Downloaded music lands in your **download folder**, not straight in the library.
From there it is [imported](./importing.md), either by hand or automatically with the
download folder watcher, and organized into your library.

## What you can do

- **Search** a service for albums, artists or tracks.
- **Paste a link** to a track, album, playlist or artist from the service.
- **Browse** an album's track list or an artist's albums before downloading.
- See the service's **charts** (top tracks), if the plugin supports it.
- **Download** a single track, a full album, an artist's whole discography, a selection
  of tracks, or a playlist.
- Check whether your **account and credentials** for the service are working.

## Choosing a service

Each configured plugin appears under **Downloaders** in the sidebar, with its icon. Click
one to open the Download section for that service. The first plugin in your
configuration is the default.

Not every service supports everything. Each plugin tells Soulsolid what it can do
(searching, searching artists, pasting links, charts), and the Download section only
shows the options the selected service supports.

### Account status

The service panel shows who you're logged in as and the state of your credentials:

| Status | Meaning |
|--------|---------|
| **Valid** | Everything is working. |
| **Invalid credentials** | Your login or API key was rejected or has expired. Update it in the plugin's `config` and restart. |
| **Disabled** | The plugin is loaded but not active. |

If no plugin is configured, the Download section tells you so and links to the setup
instructions.

## Searching

Type in the search box and pick what you're looking for: **Albums**, **Artists**,
**Tracks** or **Link**. Searches return 20 results by default.

With **Link**, paste a URL from the music service. Soulsolid works out whether it
points to a track, an album, a playlist or an artist and shows the matching result,
ready to download.

## Downloading

Downloads run in the background as [jobs](./jobs.md), so you can keep using the app
while they run. Starting a download shows a confirmation, and you can follow its
progress (and read its log) in the **Jobs** section.

Each downloaded file is:

- **Tagged** with the metadata provided by the service (title, artists, album, year,
  genre, track and disc number, ISRC and more).
- Given **cover art**, embedded in the file if artwork embedding is enabled.
- Saved with a **safe filename**, with characters that aren't allowed on disk removed.

When a download finishes, the files are in your download folder waiting to be
[imported](./importing.md).

## Configuration

```yaml
downloadPath: /app/downloads   # where downloaded files are saved
downloaders:
  plugins:
    - name: dummy
      url: https://github.com/contre95/soulsolid-dummy-plugin  # build from git, or
      # path: /app/plugins/dummy/plugin.so                     # use a pre-built plugin
      icon: https://demo2.contre.io/img/galaxy.png             # shown in the UI
      config: {}        # settings for the plugin itself, e.g. API keys
  artwork:
    embedded:
      enabled: true     # embed cover art into downloaded files
      size: 1000        # maximum artwork size in pixels
      quality: 85       # JPEG quality (0-100)
```

See [Downloading](../downloading.md) for installing plugins.

## Related

- [Downloading setup](../downloading.md): installing and configuring plugins.
- [Importing](./importing.md): moves downloaded files into the library.
- [Jobs](./jobs.md): follow and cancel running downloads.
- [Plugins](../plugins.md): writing your own downloader plugin.
