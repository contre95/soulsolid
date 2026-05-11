# Lyrics

Soulsolid supports automatic lyric retrieval and embedding via  lyric providers

Lyric handling is separate from downloaders, allowing new providers to be added without modifying downloader plugins.

## Lyric Providers

Lyric providers are responsible for searching, retrieving, and returning lyrics for tracks based on available metadata (e.g. title, artist, album, ISRC, duration).

Providers may supply:
	•	Unsynced lyrics — plain text lyrics
	•	Synced lyrics — time-aligned lyrics (e.g. LRC-style timestamps)

## Configuration

Lyric providers are configured in the main config.yaml file:

```yml
lyrics:
  providers:
    lrclib:
      enabled: true
      prefer_synced: false
```

## Provider Options

- **enabled**: Enables or disables the provider entirely.
- **prefer_synced**: When set to true, synced (LRC timestamped) lyrics will be preferred over plain text lyrics if both are available.

## Lyrics Analysis Job

The lyrics analysis job runs through all tracks in the library and attempts to fetch lyrics for each one. It can be started from the web UI under Analyze → Lyrics.

### Job Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `provider` | string | — | Which provider to use (e.g. `lrclib`) |
| `skip_existing` | bool | `false` | Skip tracks that already have lyrics |
| `override_no_queue` | bool | `false` | Apply new lyrics immediately without queuing for review, even when they differ from existing lyrics |

### Behavior

- Tracks with no lyrics: fetched and embedded directly
- Tracks with existing lyrics (when `skip_existing: false`): handled according to `override_no_queue`
  - `false` (default): new lyrics go into the queue for manual review
  - `true`: existing lyrics are silently replaced

## Lyrics Queue

When the analysis encounters tracks that need manual review, items are placed in the lyrics queue. Queue items persist in memory only and are lost on restart.

### Queue Types

| Type | Trigger | Available Actions |
|------|---------|-------------------|
| `existing_lyrics` | Track already has lyrics and new lyrics differ | `override` (accept new), `keep_old` (discard new) |
| `lyric_404` | Provider returned no lyrics for the track | `no_lyrics` (mark track as having no lyrics) |
| `failed_lyrics` | Provider search returned an error | `skip`, `edit_manual`, `no_lyrics` |
