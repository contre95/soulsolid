---
weight: 40
title: "Metadata / Tagging"
description: "Editing track tags, fetching metadata from providers, and identifying tracks with AcoustID."
icon: "sell"
draft: false
toc: true
---

The **tag editor** is where you view and fix a track's tags. You can edit them by hand,
or pull metadata from online providers (**MusicBrainz**, **Discogs** and **Deezer**) and
review it before saving. Soulsolid can also identify tracks by how they *sound* using
**AcoustID**, either one track at a time or for your whole library.

## What you can do

- **View and edit** a track's tags.
- **Fetch metadata** from MusicBrainz, Discogs or Deezer, or **search** them and pick the
  right match.
- Mark a track as **instrumental**.
- **Fingerprint** a track and look up its AcoustID.
- Run an **AcoustID analysis** over the whole library.

## The tag editor

Open the tag editor from the **Edit Tags** button on any track in the
[Library](./library.md), from the [import queue](./importing.md#the-review-queue), or from
a failed track in a job log.

The editor shows the tags **as they are in the audio file right now**, so what you see
is what's actually on disk. You can edit:

- Title and version (remix, live version, …)
- Artists and album artist
- Album, year and genre
- Track number and disc number
- ISRC, composer, BPM and replay gain
- Lyrics

Artists and albums are picked from the ones already in your library. The editor also
shows the cover art embedded in the file and lets you play or download the track.

Click **Save Changes** to write your changes both to the audio file and to the library.

### Instrumental tracks

Check **Instrumental** for tracks that have no lyrics. This clears any lyrics and tells
Soulsolid to skip the track when [fetching lyrics](./lyrics.md), so it won't keep
showing up as "lyrics not found".

## Getting metadata from providers

The provider buttons at the top of the editor (**MusicBrainz**, **Discogs**, **Deezer**)
search that service using the track's current artist, album, title and year. Only
providers enabled in your configuration are shown.

The results appear in a list showing each match's artist, album, year, genre and ISRC.
Pick the right one and the form is filled in with its data. Information that belongs to
the file itself (like its audio properties) is kept. If the match comes with an artist
or album that isn't in your library yet, it's created for you.

Fields filled in by a provider are highlighted in that provider's color, so you can see
where each value came from. **Nothing is saved until you click Save Changes**, so you
can review or adjust the values first.

## AcoustID

[AcoustID](https://acoustid.org/) identifies recordings by their sound rather than their
tags. Soulsolid computes an audio **fingerprint** of the file (using
[Chromaprint](https://acoustid.org/chromaprint)) and looks it up in the AcoustID
database. This is useful for files with wrong or missing tags, and it's what the
[Merge](./merge.md) section uses to find duplicate tracks.

From the tag editor, the **AcoustID** button fingerprints the current track and stores
its AcoustID. You can also view the stored fingerprint.

AcoustID lookups need a free client key: create an application at
[acoustid.org/new-application](https://acoustid.org/new-application) and add it to your
configuration (see below).

### AcoustID analysis

To identify the whole library at once, go to **Analyze → Metadata** and start an
**AcoustID Analysis**. It runs in the background as a [job](./jobs.md) and:

- fingerprints and looks up every track in the library,
- skips tracks that already have an AcoustID,
- reports how many tracks were processed, and how many AcoustIDs and fingerprints were
  added.

You can cancel it at any time from the Jobs section. Afterwards, use the **AcoustID**
filter in the [Library](./library.md#filters) to find tracks that still couldn't be
identified.

## Configuration

```yaml
metadata:
  providers:
    acoustid:
      enabled: true
      # secret: !env_var ACOUSTID_CLIENT_KEY   # your acoustid.org application key
    deezer:
      enabled: true
    discogs:
      enabled: true
      # secret: !env_var DISCOGS_API_KEY       # https://www.discogs.com/settings/developers
    musicbrainz:
      enabled: true
```

A disabled provider doesn't show up in the tag editor. See
[Configuration](../config.md#environment-variables) for keeping keys out of your
config file.

## Related

- [Lyrics](./lyrics.md): fetching lyrics from the tag editor or for the whole library.
- [Importing](./importing.md): fix missing tags on queued tracks here.
- [Merge](./merge.md): find duplicates once your tracks have AcoustIDs.
- [Reorganize](./reorganize.md): move files to match their corrected tags.
