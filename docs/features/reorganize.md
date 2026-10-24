---
weight: 60
title: "Reorganize"
description: "Moving library files to match your current folder layout."
icon: "folder_managed"
draft: false
toc: true
---

**Reorganize** moves the files already in your library so they match your current
[path templates](../paths.md). When you change how folders and files are named, or fix
the tags of a bunch of tracks, the files on disk are still in their old place.
Reorganize works out where each track *should* be and moves it there.

## What you can do

- **Move every misplaced file** to the location your path templates give it.
- Optionally make every path **FAT32-safe**, for syncing to SD cards, USB sticks and
  similar devices.
- See your current path templates, with a link to change them in **Settings**.

## Running a reorganize

Go to **Analyze → File Paths** and start the job. It runs in the background as a
[job](./jobs.md), so you can follow its progress and cancel it from the Jobs section.

For each track in your library, Soulsolid:

1. Works out where the file should be, using your current path templates and the
   track's tags.
2. **Skips** the track if it's already in the right place, or if its file is missing
   from disk.
3. Otherwise **moves the file** to its new location, creating folders as needed, and
   updates the library to point to it.

Folders left empty after files are moved out are removed. A track that can't be moved
never stops the job: it's counted as an error and logged, and the job carries on. When
it finishes, the job reports how many files were **moved**, **skipped** and **failed**.

> Reorganize **moves files on disk**. If other programs (a media server, a sync tool)
> point at the old locations, they'll need to rescan afterwards. A
> [webhook](./jobs.md#webhooks) can do this for you.

## When to use it

- After **changing your path templates** in Settings.
- After **correcting tags** on many tracks, for example after fixing an artist's name or
  [merging](./merge.md) duplicate artists or albums. Merging doesn't move files, so run
  a reorganize afterwards.

## FAT32-safe filenames

Check **FAT32-safe filenames** before starting the job to also clean every path so it
works on FAT32 drives:

- Everything is **lowercased**.
- Characters FAT32 doesn't allow (`: * ? " < > | \`) are replaced with `-`.
- Trailing dots and spaces are removed.
- Overly long names are shortened.
- If two tracks would end up at the same path, a number is added to the second one
  (`_1`, `_2`, …) so neither is lost.

This checkbox applies only to that run. To have every import use FAT32-safe paths too,
enable **FAT32-safe filenames** in Settings. See
[Paths → FAT32 Safe Mode](../paths.md#fat32-safe-mode) for details.

## Configuration

Reorganize has no settings of its own. It uses the same path templates as
[importing](./importing.md), under `import.paths`. See [Paths](../paths.md) for the
template syntax.

## Related

- [Paths](../paths.md): the path template syntax.
- [Importing](./importing.md): uses the same templates when files first arrive.
- [Merge](./merge.md): run a reorganize after merging.
- [Jobs](./jobs.md): follow and cancel the reorganize job.
