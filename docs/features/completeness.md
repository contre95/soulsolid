---
weight: 66
title: "Album Completeness"
description: "Finding albums that are missing tracks."
icon: "task_alt"
draft: false
toc: true
---

An album you downloaded or ripped may be missing a few tracks. **Album Completeness**
compares the tracks you have with the album's official track count and tells you which
albums are incomplete.

## What you can do

- See on every album page whether you have all of its tracks (**Complete**, or
  **Missing 3 · 9 of 12 tracks**).
- List every incomplete album in one place, the ones missing the most tracks first.
- Look an album up again, or type its track count by hand when the providers get it wrong.

Open it from **Completeness** in the sidebar (under Analyze).

## How it works

For each album, Soulsolid asks the enabled [metadata providers](./metadata.md) for the
album's official track count, trying **Deezer** first and then **MusicBrainz**. It
compares that number with the tracks in your library:

| Result | Meaning |
|--------|---------|
| **Complete** | You have at least as many tracks as the official release. Extra tracks (bonus tracks, another edition) still count as complete. |
| **Missing N** | You have fewer tracks than the official release. |
| **Not checked** / **No matching release found** | No track count is known yet, so nothing is reported. |

An album is only matched when its **title and artist match exactly** (ignoring case and
punctuation). A near match is more likely a different edition than yours, so it is skipped
rather than guessed. On MusicBrainz, if an album has several releases, the most common
track count among the official ones is used. If the album has a MusicBrainz release group
ID or a barcode, that is used to find the release instead.

> **Editions differ.** A deluxe, remastered or regional edition can have a different
> number of tracks than the one in your library. If the number looks wrong, use
> **Set count** on the album page to enter the right one.

## Checking your library

Open **Completeness** and pick one:

- **Check new albums** looks up every album that was never checked. Albums you already
  checked, and albums with a count you set by hand, are skipped.
- **Re-check all albums** looks up everything again. This replaces counts you set by hand.

The check runs as a [job](./jobs.md) (`analyze_completeness`). Lookups are spaced out to
respect the providers' rate limits, so a big library takes a while: about one album per
second. The job summary tells you how many albums were checked, how many are incomplete,
and how many had no match. Albums without a match are remembered, so they aren't retried
until you run **Re-check all albums**.

## On an album page

Open an album in the [Library](./library.md) and look under its title:

- **Check** / **Re-check** looks that one album up right away.
- **Set count** lets you type the official number of tracks. It is kept until you re-check
  the album, and the job never replaces it unless you choose **Re-check all albums**.

Clicking an album in the incomplete list opens it in the library.

## Related

- [Metadata / Tagging](./metadata.md): the providers used for the lookups.
- [Library](./library.md): where the album pages live.
- [Jobs](./jobs.md): follow completeness jobs.
