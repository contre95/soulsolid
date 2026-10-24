---
weight: 65
title: "Lyrics"
description: "Lyrics providers, analysis and the lyrics queue."
icon: "article"
draft: false
toc: true
---

Soulsolid can find lyrics for your tracks and embed them in the audio files, using lyrics providers. Lyrics providers are separate from downloader plugins, so you get lyrics no matter where your music came from.

For how to fetch lyrics and review them in the app, see [Lyrics](features/lyrics.md).

## Lyric Providers

A lyrics provider looks up lyrics for a track using its tags (title, artist, album). Currently [LRCLIB](https://lrclib.net) is supported.

Providers may supply:

- **Unsynced lyrics**: plain text lyrics
- **Synced lyrics**: lyrics with timestamps for each line (LRC format), which some players use to scroll along with the song

## Configuration

Lyrics providers are configured in `config.yaml`:

```yml
lyrics:
  providers:
    lrclib:
      enabled: true
      prefer_synced: false
```

## Provider Options

- **enabled**: Enables or disables the provider entirely.
- **prefer_synced**: When `true`, synced (timestamped) lyrics are used instead of plain lyrics when both are available. Some players, like Navidrome, need synced lyrics.

## Lyrics Analysis Job

The lyrics analysis job runs through all tracks in the library and attempts to fetch lyrics for each one. It can be started from the web UI under Analyze → Lyrics.

### Options

| Option | Default | Description |
|--------|---------|-------------|
| **Provider** | — | Which provider to use (e.g. LRCLIB) |
| **Skip tracks that already have lyrics** | on | Only look up tracks that don't have lyrics yet |
| **Override without queuing** | off | Replace existing lyrics straight away, without sending them to the queue for review |

### Behavior

- Tracks with no lyrics: lyrics are fetched and embedded directly.
- Tracks marked as instrumental: skipped.
- Tracks with existing lyrics (when not skipped):
  - by default, if the new lyrics differ, the track goes to the queue for review;
  - with **Override without queuing**, the existing lyrics are replaced.

## Lyrics Queue

When the analysis finds tracks that need your decision, they're placed in the lyrics queue. The queue is kept in memory, so it's emptied when Soulsolid restarts.

### Queue Types

| Type | Trigger | Available Actions |
|------|---------|-------------------|
| **Existing lyrics** | Track already has lyrics and the new lyrics differ | **Override** (use the new lyrics), **Edit Manually** |
| **Not found** | Provider has no lyrics for the track | **No Lyrics** (mark the track as instrumental), **Edit Manually** |
| **Failed** | Provider search returned an error | **Skip**, **No Lyrics**, **Edit Manually** |
