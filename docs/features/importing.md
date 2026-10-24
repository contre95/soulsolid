---
weight: 30
title: "Importing"
description: "Importing folders of music, duplicate handling, and the review queue."
icon: "drive_folder_upload"
draft: false
toc: true
---

**Importing** is how music gets into your library. Point Soulsolid at a folder and it
reads each file's tags, checks whether you already have the song, organizes the file
into your library folder and adds it to the library. Anything that needs your decision,
such as a duplicate or a track with missing tags, is held in a **review queue** instead
of being imported blindly.

> This page explains how importing works from the app. For every import setting
> (duplicate policy, path templates, FAT32 options) see the
> [Importing configuration guide](../importing.md) and [Paths](../paths.md).

## What you can do

- **Import a folder**, including all its subfolders.
- Let Soulsolid **watch your download folder** and import new music automatically.
- **Review queued tracks**: duplicates, tracks with missing tags, and failed imports.
- Act on the queue **one track at a time** or **a whole artist or album at once**.
- **Clean up the download folder** once you're done.

## Importing a folder

In the **Import** section, enter the folder to import (it defaults to your download
folder) and start the import. It runs in the background as a [job](./jobs.md), so you
can follow its progress and log in the **Jobs** section.

Only **MP3** and **FLAC** files are imported; anything else in the folder is ignored.

For each file, Soulsolid:

1. **Reads its tags** and computes an audio fingerprint.
2. **Checks for duplicates**: is the same recording already in your library?
3. **Checks for missing tags**: does it have an artist, album, title, year and genre?
4. Either **imports it** or **sends it to the review queue**.

When a track is imported, its file is copied (or moved, if you enabled *Move files
instead of copying*) into your library folder, at a location built from your
[path templates](../paths.md). Artists and albums are created automatically.

When the job finishes it reports how many tracks, albums and artists were imported,
and how many were skipped, queued or failed.

### Re-importing

Importing the same folder twice is safe. Tracks already in the library at the same
location are skipped, and a track that was imported before from somewhere else is
updated in place instead of being added a second time.

## Watching the download folder

Turn on **Watch Mode** in the Import section to have Soulsolid watch your download
folder. When a new file appears there, Soulsolid waits for any running job to finish
(for example, the download that's still writing files) and then imports the download
folder automatically.

To have the watcher start every time Soulsolid starts, set
`import.auto_start_watcher: true`.

## The review queue

The review queue holds tracks that need your decision. A track can be in the queue for
more than one reason at once, and each reason is shown as a badge on it:

| Reason | Why it's queued | What you can do |
|--------|-----------------|-----------------|
| **Manual review** | You enabled *Always queue tracks for review*. | Import, Cancel, Delete |
| **Duplicate** | The same recording is already in your library. | Replace, Skip, Delete |
| **Missing metadata** | A required tag is missing (see [Allow missing metadata](../importing.md#configuration-options)). | Cancel, Delete |
| **Failed import** | Something went wrong importing it, e.g. the file couldn't be read. | Skip, Delete |

What each action does:

- **Import**: adds the track to the library.
- **Replace**: replaces the copy already in your library with this file. The existing
  library entry is kept and updated, so it stays in your playlists.
- **Cancel** / **Skip**: removes the track from the queue and leaves the file where it
  is.
- **Delete**: removes the track from the queue and deletes the file from disk.

You can play a queued track and see its cover art right from the queue to help you
decide.

> **Missing tags block the library.** While a track is missing a required tag, Import
> and Replace are disabled, so an incomplete track can never be added to your library or
> overwrite one that's already there. Fix its tags in the [tag editor](./metadata.md)
> first.

### Grouping and bulk actions

Use the buttons at the top of the queue to switch between the individual view and
grouping **by artist** or **by album**. In a group you can apply an action to every
track at once. Bulk actions only touch the tracks they make sense for: *Import* skips
duplicates (use *Replace* for those), and *Replace* only touches duplicates.

**Clear Queue** empties the whole queue.

> The queue is kept in memory, so it's **emptied when Soulsolid restarts**. The files
> themselves are not touched; import the folder again to queue them again.

## Cleaning up the download folder

After importing, **Prune Download Path** (in the Import section) deletes the MP3 and
FLAC files left in your download folder and clears the review queue. Use it when you've finished reviewing and want to
start fresh.

## Configuration

```yaml
import:
  move: false              # copy files into the library (false) or move them (true)
  always_queue: false      # send every track to the review queue
  duplicates: queue        # queue | skip | replace
  allow_missing_metadata:  # per tag: true = fill in a default, false = send to the queue
    artist: false
    album: false
    title: false
    year: false
    genre: false
  auto_start_watcher: false
  paths:
    default_path: '%asciify{$albumartist}/%asciify{$album}/%asciify{$track $title}'
    fat32_safe: false
downloadPath: /app/downloads   # the folder that's watched and pruned
```

All of these can also be changed from the **Settings** page. See the
[Importing configuration guide](../importing.md) for what each option does.

## Related

- [Importing configuration](../importing.md) and [Paths](../paths.md): every import option and the path template syntax.
- [Metadata](./metadata.md): fix missing tags before importing a queued track.
- [Downloading](./downloading.md): produces the files the watcher picks up.
- [Jobs](./jobs.md): follow and cancel imports.
