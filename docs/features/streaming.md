---
weight: 80
title: "Streaming"
description: "Playing your music in the browser."
icon: "graphic_eq"
draft: false
toc: true
---

Soulsolid has a built-in audio player so you can **listen to a track right in the
browser** while you browse, tag or review your music. It's there to help you check
what you're looking at (is this the right version? is the file broken?) rather than to
be a full music player.

## Where you can play music

A player appears in:

- the **track overview** panel in the [Library](./library.md#track-overview),
- the [tag editor](./metadata.md), which also lets you download the file,
- the [import queue](./importing.md#the-review-queue), so you can listen to a track
  before deciding whether to import or replace it.

You can skip to any point in the track.

## Supported formats

Soulsolid plays the files as they are, with no conversion, so playback depends on what
your browser supports. **MP3** and **FLAC** are the main formats Soulsolid works with and
play in all modern browsers. WAV, AAC, M4A, OGG, Opus and WMA files can be played too,
if your browser supports them.

## Privacy and safety

Only files inside your **library folder** or your **download folder** can be played.
Requests for any other file on the server are refused, so the player can't be used to
read files outside your music folders. Only audio files are served.

## Configuration

Streaming has no settings of its own. It plays files from:

```yaml
libraryPath: /app/music       # your library
downloadPath: /app/downloads  # your downloads
```

## Related

- [Library](./library.md): play tracks from the track overview.
- [Importing](./importing.md): listen to queued tracks before deciding.
- [Playlists](./playlists.md): export playlists to play them in other players.
