---
weight: 65
title: "Merge"
description: "Cleaning up duplicate artists, albums, genres and tracks."
icon: "merge"
draft: false
toc: true
---

Over time a library collects the same artist, album or genre spelled in slightly
different ways: `Hip-Hop`, `Hip Hop` and `hiphop`, or `Un Verano Sin Tí` and
`un verano sin ti`. It also collects the same song stored twice, say a FLAC and an MP3
of the same recording. The **Merge** section finds these and lets you clean them up.

## What you can do

- Find **artists**, **albums** and **genres** that are the same but spelled differently,
  and merge them into one.
- Find **duplicate tracks** (the same recording in more than one file) and delete the
  copies you don't want.
- Mark a group of duplicate tracks as **intentional**, so it stops showing up.

Open it from **Merge** in the sidebar (under Analyze).

## Artists, albums and genres

Click **Scan** on the Artists, Albums or Genres card. Soulsolid groups names that only
differ by **case, punctuation, accents or spacing**. Albums are only grouped when their
artist matches too, so two different albums that share a title aren't mixed up.

For each group:

1. Pick the name to **keep** with the radio button. Soulsolid suggests the
   best-looking one (proper capitalization over ALL CAPS or all lowercase).
2. Untick any variant you *don't* want merged.
3. Click **Merge**.

The merge runs as a [job](./jobs.md). Every track that used one of the other spellings
is moved over to the name you kept, and its **audio file tags are updated** too, so
the fix sticks even outside Soulsolid.

> **File paths aren't updated after a merge.** If your folder names include the artist,
> album or genre, run [Reorganize](./reorganize.md) afterwards to move the files to
> their new folders.

## Duplicate tracks

Duplicate tracks are found by **AcoustID**: two files with the same AcoustID are the
same recording, even if they're in different formats, bitrates or have slightly
different tags. So run an [AcoustID analysis](./metadata.md#acoustid-analysis) first; tracks
without an AcoustID can't be matched.

> Why not on import? When importing, Soulsolid compares the audio fingerprints
> directly, which already catches most duplicates without depending on an online
> service. But small differences (another encoding, bitrate, a slight edit, noise) can
> make two fingerprints of the same song differ slightly. AcoustID, a crowdsourced ID
> for each recording, catches those.

Click **Scan** on the Tracks card. For each group of duplicates you have two choices:

- **Delete selected**: tick the files you want to get rid of and click it. The ticked
  files are **permanently deleted from disk** and removed from the library. Playlists
  that contained a deleted copy now point to the copy you kept.
- **Keep**: leaves every file where it is and records the group as intentional
  duplicates, so it stops showing up in future scans. Nothing on disk changes.

Marked a group as *Keep* by mistake? Tick **Include kept** before scanning to see those
groups again.

## Related

- [Metadata / Tagging](./metadata.md): run an AcoustID analysis before looking for duplicate tracks.
- [Reorganize](./reorganize.md): move files to match their merged names.
- [Importing](./importing.md#the-review-queue): how duplicates are handled when music first arrives.
- [Jobs](./jobs.md): follow merge jobs.
