---
weight: 50
title: "Lyrics"
description: "Fetching lyrics for one track or the whole library, and reviewing conflicts."
icon: "lyrics"
draft: false
toc: true
---

Soulsolid can find lyrics for your tracks and save them into the audio files, so any
player that reads embedded lyrics can show them. You can fetch lyrics for **one track**
from the tag editor, or run a **lyrics analysis** over your whole library. Because
lyrics you've curated by hand are easy to lose, Soulsolid never overwrites existing
lyrics without asking: anything unclear goes to a **review queue**.

Lyrics come from **lyrics providers**. Currently [LRCLIB](https://lrclib.net) is
supported, which offers both plain and synced (time-stamped) lyrics. See
[Lyrics configuration](../lyrics.md) for provider settings.

## What you can do

- **Fetch lyrics for a single track** in the tag editor.
- **Analyze the whole library** and add lyrics to every track that's missing them.
- **Review** tracks where the new lyrics differ from the existing ones, or where no
  lyrics were found.
- Mark tracks as **instrumental** so they're skipped from now on.

## Lyrics for one track

In the [tag editor](./metadata.md), the **Lyrics** buttons fetch lyrics from each enabled
provider and put them in the lyrics field. Review them and click **Save Changes** to
write them to the file.

## Lyrics analysis

Go to **Analyze → Lyrics**, choose a provider and click **Start Lyrics Analysis**. It
runs in the background as a [job](./jobs.md) that goes through every track in your
library. You can follow its progress and cancel it from the Jobs section.

Two options control what happens to tracks that **already have lyrics** (you can pick
one or neither):

| Option | Behavior |
|--------|----------|
| **Skip tracks that already have lyrics** *(on by default)* | Tracks with lyrics are left alone; only tracks without lyrics are looked up. |
| **Override without queuing** | New lyrics replace existing ones straight away, without asking. |
| *Neither* | When the new lyrics differ from the existing ones, the track goes to the review queue so you can decide. |

For every track:

- **Instrumental** tracks are skipped.
- If the track has **no lyrics**, the ones found are saved directly.
- If the new lyrics are **identical** to the existing ones, nothing changes.
- If **no lyrics are found**, or the provider **fails**, the track goes to the review
  queue.

A track that fails never stops the analysis. When it finishes, the job reports how many
tracks were updated, skipped, queued or failed. In the [job log](./jobs.md#job-logs), each
failed track has a link to open it in the tag editor and fix it by hand.

## The lyrics queue

The lyrics queue, under **Analyze → Lyrics**, holds the tracks that need your decision:

| Reason | What happened | What you can do |
|--------|---------------|-----------------|
| **Existing lyrics** | New lyrics were found that differ from the ones already on the track. Compare the old and new versions. | **Override** with the new lyrics, or **Edit Manually** |
| **Not found** | The provider has no lyrics for this track. | **No Lyrics** (mark as instrumental), or **Edit Manually** |
| **Failed** | The provider returned an error. | **Skip**, **No Lyrics**, or **Edit Manually** |

*Edit Manually* opens the track in the [tag editor](./metadata.md), where you can paste or
fetch lyrics yourself.

As with the [import queue](./importing.md#grouping-and-bulk-actions), you can group items
**by artist** or **by album** and act on a whole group at once, or clear the queue.

> The lyrics queue is kept in memory, so it's **emptied when Soulsolid restarts**. Run
> the analysis again to rebuild it.

## Where lyrics are saved

Lyrics are written to the **audio file's tags** and to the library, so you can
[search inside lyrics](./library.md#filters) and see them in the track overview.

## Configuration

```yaml
lyrics:
  providers:
    lrclib:
      enabled: true
      prefer_synced: false  # prefer time-stamped lyrics when available
```

See [Lyrics configuration](../lyrics.md) for details.

## Related

- [Lyrics configuration](../lyrics.md): providers and their options.
- [Metadata / Tagging](./metadata.md): the tag editor, and marking tracks as instrumental.
- [Jobs](./jobs.md): follow and cancel the lyrics analysis.
- [Library](./library.md): filter tracks by lyrics.
