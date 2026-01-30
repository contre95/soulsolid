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
- **prefer_synced**: When set to true, synced lyrics will be preferred over unsynced lyrics if both are available.
