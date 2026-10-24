---
weight: 70
title: "Playlists"
description: "Creating, editing, and exporting playlists built from library tracks."
icon: "playlist_play"
draft: false
toc: true
---

**Playlists** let you group tracks from your library into named, ordered lists and
export them as standard `.m3u` files that other players and devices can open.

## What you can do

- **Create** a playlist with a name and an optional description.
- **Add** a single track, a whole album, or every track by an artist.
- **Remove** tracks from a playlist.
- **Rename**, edit the description of, or **delete** a playlist.
- **Export** a playlist as an `.m3u` file.

## Creating a playlist

In the **Playlists** section, click to create a new playlist, give it a name and,
optionally, a description.

## Adding music

From the [Library](./library.md), use the **Add to Playlist** button on any track, album or
artist and pick the playlist.

- Adding a **track** adds just that track.
- Adding an **album** adds every track on it.
- Adding an **artist** adds every track by that artist.

Tracks that are already in the playlist are skipped, so adding an album twice won't
duplicate anything.

## Managing a playlist

Open a playlist to see its tracks with their title, artist and duration, and the total
duration of the playlist. From there you can remove tracks, edit the name and
description, or delete the playlist.

Deleting a playlist doesn't delete any music: the tracks stay in your library.

## Exporting to M3U

**Export** downloads the playlist as `<playlist name>.m3u`. The file lists each track
in playlist order with its duration, artist and title, followed by the full path to the
audio file on disk.

Because the file uses full paths, it works best with players that see your music at the
same location as Soulsolid, such as a media server reading the same library folder.

## Configuration

Playlists have no settings. They're stored in the library database, so they're
included when you [back up the database](./config.md#backing-up-the-database).

## Related

- [Library](./library.md): where you add tracks, albums and artists to playlists.
- [Streaming](./streaming.md): play your tracks in the browser.
- [Downloading](./downloading.md): download playlists from a music service.
