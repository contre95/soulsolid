---
weight: 40
title: "Importing"
description: "Importing music, duplicate handling and the import queue."
icon: "article"
date: "2025-09-09T21:10:30+02:00"
lastmod: "2025-09-09T21:10:30+02:00"
draft: false
toc: true
---
Soulsolid imports music from folders on the server. When importing, it reads each file's tags, checks whether you already have the song, moves or copies the file into your library folder, and creates artists and albums as needed.

This page covers the import settings. For how to import and review tracks in the app, see [Importing](features/importing.md).

## Configuration Options

Import behavior is controlled in `config.yaml` (most options can also be changed from the **Settings** page):

```yaml
import:
  move: false           # if false, files are copied; if true, originals are removed after import
  always_queue: false   # queue every track for manual review, even non-duplicates
  duplicates: queue     # queue | skip | replace
  allow_missing_metadata:        # per-field control over importing tracks with missing metadata
    artist: false                # when true, a missing field is filled with a fallback default on import;
    album: false                 # when false, a track missing that field is sent to the missing_metadata queue
    title: false
    year: false
    genre: false
  auto_start_watcher: false      # automatically watch the download path on startup
  paths:
    compilations: '%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:soundtrack: '%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:single: '%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:ep: '%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    default_path: '%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    fat32_safe: false     # lowercase paths, strip FAT32-forbidden characters, 255-byte segment limit
```


## Import Sources

### Directory Import

Imports every supported audio file in a folder and its subfolders. This is the most common way to bring an existing music collection into Soulsolid.

**Supported file formats:**
- MP3 (.mp3)
- FLAC (.flac)

**Process:**
1. Finds every supported audio file in the folder and its subfolders
2. Reads the tags of each file
3. Computes an audio fingerprint to detect duplicates
4. Copies or moves the file into your library folder
5. Creates artists and albums as needed
6. Adds the track to your library

### Download Path Watcher

The watcher keeps an eye on your `downloadPath` folder. When a new audio file appears there, it waits for any running job to finish (up to 5 minutes, so a download can finish writing its files) and then imports the download folder automatically.

- Start it automatically with Soulsolid: `import.auto_start_watcher: true`
- Turn it on or off at any time with **Watch Mode** in the Import section
- Only new files start an import; changing or deleting files doesn't

## File Organization

### Move vs Copy

The import behavior can be configured to either move or copy files:

- **Move**: Files are moved from their original location to the library structure
- **Copy**: Files are copied, leaving originals in place

### Library Structure

Files are organized into folders according to your [`import.paths` templates](paths.md).
The following config:
```yaml
  paths:
    default_path: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
```
Will produce the following files structure:
```
Library/
├── Artist Name/
│   ├── Album Title (Year)/
│   │   ├── 01 - Track Title.mp3
│   │   ├── 02 - Track Title.mp3
│   │   └── ...
│   └── ...
└── Various Artists/
    └── Compilation Album (Year)/
        └── ...
```


### Special Handling

- **Compilations**: Multi-artist albums are placed under `Various Artists`
- **Path Customization**: Different path patterns for singles, EPs, soundtracks, etc. See [Paths](paths.md) for examples.
- **Filename Sanitization**: Characters that aren't allowed in file names are removed

### FAT32-Safe Filenames

When `fat32_safe: true` is set, every folder and file name is made FAT32-compatible on import:

- Everything is lowercased, so names don't clash when syncing to FAT32 (where `Rock/` and `rock/` are the same folder)
- Characters FAT32 doesn't allow (`: * ? " < > | \`) are replaced with `-`
- Trailing dots and spaces are removed
- Names longer than 255 bytes are shortened, keeping the file extension

If two tracks would resolve to the same lowercased path (e.g. `Wos/lala.mp3` and `WoS/lala.mp3`), a number is added to the second (`wos/lala_1.mp3`) so both files are kept.

## Duplicate Detection

Duplicates are detected with [Chromaprint](https://acoustid.org/chromaprint) audio fingerprints, which identify the same recording by how it sounds, regardless of its file name or tags.

To find duplicates already in your library, see [Merge](features/merge.md).

## Duplicate Handling Strategies

When duplicates are detected, the system can be configured to:

### Skip
- Ignores duplicate tracks completely
- No changes made to existing tracks
- Fastest option for large imports

### Replace
- Replaces the track in your library with the new file
- Updates its tags and location
- Keeps the existing library entry, so it stays in your playlists
- A new file that's missing required tags is queued instead, so it can't overwrite a complete track

### Queue
- Adds duplicate to manual review queue
- Allows user to decide how to handle each case
- Most conservative approach

### Always Queue
You can also send every track to the queue, duplicate or not, with `always_queue: true`.
- Forces all tracks (including non-duplicates) into queue
- Maximum control over import process
- Useful for carefully curated libraries

## Import Queue

The import queue holds tracks that need your approval before they're added to the library. The queue is kept in memory, so it's emptied when Soulsolid restarts (the files themselves aren't touched).

### Queue Types

A track can be in the queue for **more than one reason at once** (e.g. it's both a
duplicate and missing metadata). Each reason is shown as its own badge.

| Type | Trigger | Available Actions |
|------|---------|-------------------|
| **Manual review** | `always_queue: true` | Import, Cancel, Delete |
| **Missing metadata** | Track is missing a tag whose `allow_missing_metadata.<field>` is `false` | Cancel, Delete |
| **Duplicate** | The same recording is already in your library | Replace, Skip, Delete |
| **Failed import** | Something went wrong importing it (e.g. the file can't be read) | Skip, Delete |

**Missing metadata blocks the library.** While a track is missing a required tag, Import
and Replace are disabled (only Cancel/Skip and Delete remain), so an incomplete track can
never be added to your library or overwrite one that's already there. For example, a
duplicate that's also missing metadata can only be skipped or deleted until its tags are
fixed in the [tag editor](features/metadata.md).

See [Importing → The review queue](features/importing.md#the-review-queue) for what each action does.

### Telegram Integration

> **Deprecated:** the Telegram bot will be removed in v0.30.0. Use the web UI instead.

You can interact with the import queue directly from Telegram using the following commands:

- `/queue` - Show the current queue and process items one by one
- `/queue_clear` - Clear the entire import queue

When using `/queue`, you'll receive queue items with inline buttons to:
- **Import** or **Cancel** for manual review items
- **Replace** or **Skip** for duplicate items
- **Next** to proceed to the next item
- **Cancel Session** to end the current queue session

This allows you to manage imports remotely without needing to access the web interface.

### Queue Behavior with Move Option

With `move: true`, a queued track is removed from the queue once it's imported, since its original file has been moved into your library and is no longer where the queue expects it.

## Import Statistics

Each import job reports:
- **Imported**: tracks, albums and artists added to the library
- **Skipped**: duplicates that were ignored
- **Queued**: tracks sent to the review queue
- **Errors**: tracks that failed, with details in the job log

