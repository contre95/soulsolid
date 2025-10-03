---
weight: 999
title: "Importing"
description: ""
icon: "article"
date: "2025-09-09T21:10:30+02:00"
lastmod: "2025-09-09T21:10:30+02:00"
draft: false
toc: true
---
The soulsolid application supports importing music from two primary sources: local directories and Beets music library databases. The import process handles metadata extraction, duplicate detection, file organization, and automatic artist/album creation.

## Configuration Options

Import behavior is controlled through the application configuration:

```yaml
import:
  move: false
  always_queue: true
  duplicates: replace
  paths:
    compilations: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:soundtrack: '%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:single: '%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    album:ep: '%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}'
    default_path: '%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}'

```


## Import Sources

### Directory Import

Imports all supported audio files from a specified directory recursively. This is the most common way to import existing music collections.

**Supported file formats:**
- MP3 (.mp3)
- FLAC (.flac)

**Process:**
1. Scans directory recursively for supported audio files
2. Extracts metadata from file tags
3. Generates audio fingerprints for duplicate detection
4. Organizes files into library structure
5. Creates artists and albums as needed
6. Adds tracks to the music library

### Beets Database Import

Imports music from a Beets music library database file (.db). This is useful for migrating from Beets or importing curated collections with rich metadata.

**Requirements:**
- SQLite database file (.db) exported from Beets
- Original audio files must be accessible at their stored paths
- Maximum file size: 100MB

**Process:**
1. Reads track metadata from Beets database
2. Verifies file existence at stored paths
3. Processes tracks similar to directory import
4. Handles missing files gracefully

## File Organization

### Move vs Copy

The import behavior can be configured to either move or copy files:

- **Move**: Files are moved from their original location to the library structure
- **Copy**: Files are copied, leaving originals in place

### Library Structure

Files are organized into a hierarchical depending on the `import.path` config.
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
- **Path Customization**: Different path patterns for singles, EPs, soundtracks, etc. You can see the [default config](https://github.com/contre95/soulsolid/blob/2a07d80165471205f0498e0c0715b3a866562a74/config.yaml?plain=1#L39-L44) for reference.
- **Filename Sanitization**: Removes invalid characters and creates clean filenames

## Duplicate Detection

The system has two different methods to detect duplicate tracks:

### Primary: Audio Fingerprinting

- Uses [Chromaprint](https://acoustid.org/chromaprint) algorithm to generate unique audio fingerprints
- Compares fingerprints to identify identical audio content. This is the most reliable method for detecting true duplicates

### Fallback: Metadata Comparison

When fingerprinting fails or is unavailable:
- Compares track title, artist name, and album title. I is less reliable but still effective for most cases. It's only used when fingerprint generation encounters errors.

## Duplicate Handling Strategies

When duplicates are detected, the system can be configured to:

### Skip
- Ignores duplicate tracks completely
- No changes made to existing tracks
- Fastest option for large imports

### Replace
- Replaces existing track with new file
- Updates metadata and file path
- Preserves track ID and relationships

### Queue
- Adds duplicate to manual review queue
- Allows user to decide how to handle each case
- Most conservative approach

### Always Queue
Additionally you can opt to always queue regardless weather they are duplicates setting the `always_queue: true`.
- Forces all tracks (including non-duplicates) into queue
- Maximum control over import process
- Useful for carefully curated libraries

## Import Queue

The import queue provides manual review capabilities for tracks that require user approval before being added to the library. The queue stores tracks in memory only, so all queued items will be lost if the system is restarted.

### Queue Types
- **Manual Review**: New tracks requiring approval, typically when the `always_queue: true` option is set.
- **Duplicate**: Tracks that already exist in the library and were sent to the queue for manual review.

### Telegram Integration

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

When the `move: true` option is enabled in the import configuration, tracks in the queue are automatically removed from the queue after being imported. This prevents errors that could occur when trying to import a track whose original file no longer exists at the source location after it has been moved to the library structure.

## Import Statistics

Each import job provides detailed statistics:
- **Tracks Imported**: Successfully added to library
- **Skipped**: Duplicates that were ignored
- **Errors**: Failed imports with error details

