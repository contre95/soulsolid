---
weight: 10
title: "Library"
description: "Browsing, searching, and managing the music collection."
icon: "library_music"
draft: false
toc: true
---

The **Library** is the heart of Soulsolid: it's your music collection once it has been
imported. Everything else in the app (importing, downloading, tagging, lyrics) ends up
here, and the Library section is where you browse, search, inspect and clean it up.

## What you can do

- **Browse** every track in the collection, page by page.
- **Search** artists, albums and tracks from a single search box.
- **Filter** tracks by genre, lyrics, AcoustID or date added.
- Open a **track overview** with its tags and audio details.
- Jump to the **tag editor**, **add to a playlist**, or **play** a track.
- **Delete** tracks, albums or whole artists.
- See **collection stats**: how many artists, albums and tracks you have and how much
  disk space they use.

## Browsing and searching

With the search box empty, the Library lists every track in your collection, 50 per
page.

Start typing to search. A single query looks at artist names, album titles and track
titles at the same time. Matching **albums and artists are listed first**, followed
by matching tracks. A track matches if its title, its artist or its album contains
the text you typed.

### Filters

Filters narrow down the track results and can be combined with each other and with a
text search:

| Filter | What it does |
|--------|--------------|
| **Genre** | Only tracks with exactly that genre. |
| **AcoustID** | Only tracks that have (or don't have) an AcoustID. Useful to find tracks that still need [identifying](./metadata.md#acoustid-analysis). |
| **Lyrics** | *Has lyrics*, *Empty lyrics*, *Instrumental*, or *Lyrics contain…* to search inside the lyrics text. |
| **Added after / before** | Only tracks added to the library in a given date range. |

## Track overview

The **Overview** button opens a floating panel with everything Soulsolid knows about a
track: title, artist, album, year, genre, track number, ISRC, BPM and lyrics, plus
technical details such as format, bitrate, sample rate, bit depth, channels, duration
and the file's location on disk. You can also play the track right there (see
[Streaming](./streaming.md)).

From each search result you can also:

- **Edit Tags** to open the track in the [tag editor](./metadata.md).
- **Add to Playlist** to add the track, or a whole album or artist, to one of your
  [playlists](./playlists.md).

## Deleting music

You can delete a single track, an album, or an artist with all their tracks.

> **Deleting removes the audio files from disk**, not just the entry in the library.
> This can't be undone.

If a file can't be removed (for example it was already deleted by hand), the entry is
still removed from the library and the problem is logged.

## Collection stats

The [dashboard](./ui.md) and the Library show how many **artists**, **albums** and
**tracks** you have, and the **total size** of your library folder on disk.

The folder buttons at the top of the Library open a **folder tree** of your library or
downloads folder in a new tab, so you can check at a glance how files are laid out on
disk. This needs the `tree` command
to be installed (it is in the official container image).

## Configuration

The Library has no settings of its own. It uses:

```yaml
libraryPath: /app/music      # where your organized music lives
downloadPath: /app/downloads # shown in the downloads folder tree
database:
  path: /data/library.db     # the library database
```

## Related

- [Importing](./importing.md) and [Downloading](./downloading.md) add music to the library.
- [Metadata](./metadata.md) and [Lyrics](./lyrics.md) enrich the tracks already in it.
- [Merge](./merge.md) cleans up duplicate artists, albums, genres and tracks.
- [Metrics](./metrics.md) turns the library into charts.
