---
weight: 70
title: "Playlists"
description: "Creating, editing, and exporting playlists built from library tracks."
icon: "playlist_play"
draft: false
toc: true
---

The **playlists** feature lets you group library tracks into named, ordered
collections and export them as standard `.m3u` files. Playlists are stored in the
database and reference existing library tracks — adding an artist or an album is a
convenience that expands to all of that entity's tracks. There is no background job
here; everything is synchronous CRUD over the playlist repository.

## What it does

- **Create, rename, update, and delete** playlists (name + description).
- **Add items** to a playlist by `track`, `artist`, or `album` — artists and albums
  expand into all of their tracks, skipping duplicates already present.
- **Remove** an individual track from a playlist.
- **Export** a playlist to an `#EXTM3U` file with per-track `#EXTINF` duration/artist
  lines and absolute file paths.
- Provide HTMX **modal partials** for creating a playlist and for picking a playlist
  to add an item to.

## How it works

A standard, job-free three-layer feature backed by a dedicated repository:

```
routes.go    → registers the /playlists routes (and builds the handler)
handlers.go  → HTTP layer: form parsing, HTMX partials, M3U streaming
service.go   → domain logic over the PlaylistRepository + Library
```

- **`Service`** (`service.go`) wraps a `music.PlaylistRepository` (persistence) and
  the `music.Library` (to resolve track/artist/album names and expand entities). It
  generates IDs with `music.GeneratePlaylistID()` and validates playlists via
  `Playlist.Validate()` before create/update.
- **`Handler`** (`handlers.go`) negotiates HTMX vs. plain responses, sets
  `HX-Trigger` events (`refreshPlaylists`, `playlistUpdated`) so lists and badges
  refresh, and surfaces results through the toast system.

### Adding items (track / artist / album)

`AddItemToPlaylist` first verifies the playlist exists, then resolves the set of
track IDs to insert based on `item_type`:

- `track` → the single track (existence-checked).
- `artist` → every track by that artist, fetched via
  `GetTracksFilteredPaginated` with an `ArtistIDs` filter (large page size).
- `album` → every track on that album, via an `AlbumIDs` filter.

Each resolved track is added through `AddTrackToPlaylist`; an "already exists" error
is treated as a benign skip so re-adding never errors. The handler then composes a
type-appropriate success toast ("Track … added", "All tracks by … added", etc.).

### M3U export

Two export paths exist:

- **`ExportM3U` handler** (`GET /playlists/:id/export`) builds the `#EXTM3U`
  content in memory and streams it back with a `Content-Disposition` filename of
  `<playlist name>.m3u`, routed through `respond.Resource`.
- **`Service.ExportM3U`** writes the same content to a file path on disk (used by
  non-HTTP callers).

Both emit a `#EXTINF:<duration>,<artists> - <title>` line followed by the track's
absolute path for each track in playlist order.

## Endpoints

Registered in `src/features/playlists/routes.go`. Note that `/playlists/:id` is the
catch-all GET and is registered last so the more specific routes match first.

| Method | Route | Purpose | Response |
|--------|-------|---------|----------|
| GET | `/playlists` | Render the Playlists section | Section |
| GET | `/playlists/create-modal` | Create-playlist modal partial | Partial |
| POST | `/playlists/` | Create a playlist (`name`, `description`) | Toast |
| PUT | `/playlists/:id` | Rename / update a playlist | Toast |
| DELETE | `/playlists/:id` | Delete a playlist | Toast |
| POST | `/playlists/items` | Add an item (`playlist_id`, `item_type`, `item_id`) | Toast |
| DELETE | `/playlists/:playlistId/tracks/:trackId` | Remove a track from a playlist | Toast |
| GET | `/playlists/:type/:id/playlists` | "Add to playlist" modal for a track/artist/album | Partial |
| GET | `/playlists/:id/export` | Export the playlist as `.m3u` | Resource (M3U) |
| GET | `/playlists/:id` | Render a single playlist | Partial |

### Form fields for `POST /playlists/items`

| Field | Meaning |
|-------|---------|
| `playlist_id` | Target playlist |
| `item_type` | `track`, `artist`, or `album` |
| `item_id` | ID of the entity to add (expanded to tracks for artist/album) |

## Configuration

Playlists have no dedicated config block. The service receives the `config.Manager`
for consistency with other features but relies on the database for all state.

## Related

- [Library](./library.md) — the source of tracks, artists, and albums added to playlists.
- [Streaming](./streaming.md) — play tracks that make up a playlist.
- [API reference](../api.md) — response-type semantics (`Section`, `Partial`, `Resource`).
