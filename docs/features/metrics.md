---
weight: 90
title: "Metrics"
description: "Charts about your library: genres, years, formats and how complete your tags are."
icon: "monitoring"
draft: false
toc: true
---

**Metrics** turn your library into charts on the [dashboard](./ui.md), so you can see
what your collection is made of and how well tagged it is.

## The charts

| Chart | What it shows |
|-------|---------------|
| **Overview** | How many tracks, artists and albums you have. |
| **Release years** | How many tracks were released each year, from 1900 onwards. |
| **Genres** | How your tracks are spread across genres. Bigger tiles mean more tracks. |
| **Formats** | The share of each audio format (MP3, FLAC, …) in your library. |
| **Metadata completeness** | The percentage of tracks that have an ISRC, BPM, year, genre and lyrics. |

The **metadata completeness** chart is a quick way to see what needs work: a low
*Lyrics* bar means it's time for a [lyrics analysis](./lyrics.md#lyrics-analysis), and
low *Year* or *Genre* bars point to tracks worth fixing in the
[tag editor](./metadata.md).

## Keeping metrics up to date

Most charts are calculated once and saved, so the dashboard stays fast even with a
large library. This means they **don't update by themselves** as your library changes.

Click **Refresh Metrics** on the dashboard to recalculate them. This runs in the
background as a [job](./jobs.md); the charts show the new numbers once it finishes.
Refresh after a big import, a merge, or a lyrics analysis.

The *metadata completeness* chart is always calculated live, so it's always current.

## Configuration

Metrics have no settings.

## Related

- [UI / Dashboard](./ui.md): where the charts are shown.
- [Library](./library.md): the collection the metrics describe.
- [Lyrics](./lyrics.md) and [Metadata](./metadata.md): improve the completeness numbers.
- [Jobs](./jobs.md): the *calculate metrics* job.
