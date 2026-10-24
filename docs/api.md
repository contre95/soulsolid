---
weight: 90
title: "API"
description: "Using Soulsolid from scripts: the HTTP API reference."
icon: "article"
draft: false
toc: true
---

Everything you can do in the web interface can also be done over HTTP, which makes it
easy to automate Soulsolid with `curl`, scripts or other tools. The web interface and
the API use the **same addresses**: when a request comes from the web interface you get
a web page, and when it comes from anything else you get **JSON**.

```bash
# How many tracks are in the library?
curl http://localhost:3535/library/tracks/count
# {"key":"tracks_count","value":1234}

# Import the download folder
curl -X POST http://localhost:3535/import/directory -d "directoryPath=/app/downloads"
# {"job_id":"…"}

# Follow the job
curl http://localhost:3535/jobs/<job_id>
```

Request bodies can be sent as form fields (`-d "key=value"`) or as JSON.

> Soulsolid has **no authentication**: anyone who can reach it can use the API. Don't
> expose it to the internet without a reverse proxy that adds a login.

## Responses

Each route below returns one of these kinds of responses:

| Response | What you get |
|----------|--------------|
| **Page** | A full web page. Open it in a browser. |
| **Data** | The data shown on that part of the page, as JSON. |
| **Value** | A single value: `{"key":"…","value":…}`. |
| **OK** | `{"message":"…"}` on success, or `{"error":"…"}` with an HTTP error status. |
| **Job** | `202 Accepted` with `{"job_id":"…"}`. The work runs as a background [job](features/jobs.md); follow it with `/jobs/:id`. |
| **File** | The file itself (audio, image, playlist, database). Send `Accept: application/json` to get `{"type":"…","url":"…"}` instead. |
| **JSON** | Always JSON. |

---

## Dashboard

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/` | Dashboard | Page |
| GET | `/dashboard` | Dashboard | Page |
| GET | `/analyze` | Analyze section | Page |
| GET | `/dashboard/quick-actions` | Quick actions card | Data |
| GET | `/health` | Returns `OK` while the server is running | Text |

---

## Settings

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/settings` | Settings page | Page |
| GET | `/config/form` | Current settings | Data |
| PUT | `/settings` | Save settings | OK |
| PUT | `/config/import` | Save import options | OK |
| PUT | `/config/paths` | Save path templates | OK |
| GET | `/config` | The whole configuration | JSON |
| GET | `/config?fmt=yaml` | The `config.yaml` file as text | YAML |
| GET | `/config/database/download` | Download a backup of the library database | File |

---

## Library

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/library` | Library page | Page |
| GET | `/library/table` | Library table | Data |
| GET | `/library/search` | Search the library (see parameters below) | Data |
| GET | `/library/tracks/:trackId/overview` | Track overview | Data |
| GET | `/library/artists/count` | Number of artists | Value |
| GET | `/library/albums/count` | Number of albums | Value |
| GET | `/library/tracks/count` | Number of tracks | Value |
| GET | `/library/storage/size` | Library size on disk, in bytes | Value |
| GET | `/library/artists/:id` | One artist | JSON |
| GET | `/library/albums/:id` | One album | JSON |
| GET | `/library/tracks/:id` | One track | JSON |
| GET | `/library/tree` | Folder tree of the library (`?folder=downloads` for the download folder) | Value |
| GET | `/library/tracks/:id/lyrics` | A track's lyrics | Value |
| DELETE | `/library/tracks/:trackId` | Delete a track **and its file** | OK |
| DELETE | `/library/albums/:albumId` | Delete an album **and its files** | OK |
| DELETE | `/library/artists/:artistId` | Delete an artist **and their files** | OK |

### Search parameters

| Parameter | Default | Meaning |
|-----------|---------|---------|
| `query` | | Text to search for in artists, albums and tracks |
| `page` | `1` | Page number |
| `limit` | `50` | Results per page |
| `genre` | | Only tracks with this genre |
| `has_acoustid` | | `true` or `false`: only tracks with or without an AcoustID |
| `lyrics_filter` | | Only tracks in a given lyrics state |
| `lyrics_text` | | Only tracks whose lyrics contain this text |
| `added_after`, `added_before` | | Only tracks added in this date range |

---

## Tags / Metadata

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/tag/:trackId` | Tag editor for a track (`?source=db` to show the library's tags instead of the file's) | Page |
| POST | `/tag/:trackId` | Save a track's tags | OK |
| GET | `/tag/:trackId/metadata` | Enabled metadata providers | Data |
| GET | `/tag/:trackId/search/:provider` | Search a provider for matches | Data |
| GET | `/tag/:trackId/select/:provider?index=N` | Fill the tag editor with a search result | Page |
| GET | `/tag/:trackId/:provider` | Fill the tag editor with a provider's best match | Page |
| GET | `/tag/:trackId/artwork` | The track's cover art | File |
| GET | `/tag/:trackId/fingerprint` | Fingerprint the track and look up its AcoustID | OK |
| GET | `/tag/:trackId/fingerprint/view` | The track's stored fingerprint | Value |
| GET | `/analyze/metadata` | Metadata analysis page | Page |
| POST | `/analyze/acoustid` | Start an [AcoustID analysis](features/metadata.md#acoustid-analysis) of the library | Job |

---

## Importing

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/import` | Import page | Page |
| GET | `/import/directory/form` | Import form, with the default folder | Data |
| GET | `/import/directory/badges` | Current import options | Data |
| POST | `/import/directory` | Import a folder (`directoryPath`) | Job |
| GET | `/import/queue/items` | Tracks in the review queue | Data |
| GET | `/import/queue/items/grouped` | Queue grouped by `?type=artist` or `?type=album` | Data |
| GET | `/import/queue/header` | Queue summary | Data |
| GET | `/import/queue/count` | Number of tracks in the queue | Value |
| GET | `/import/queue/:id/artwork` | Cover art of a queued track | File |
| POST | `/import/queue/:id/:action` | Act on a queued track: `import`, `replace`, `cancel` or `delete` | OK |
| POST | `/import/queue/group/:groupType/:groupKey/:action` | Act on a whole artist or album group | OK |
| POST | `/import/queue/clear` | Empty the queue | OK |
| POST | `/import/prune/download-path` | Delete the audio files in the download folder and empty the queue | OK |
| POST | `/import/watcher/toggle` | Start or stop the download folder watcher (`action=start` or `action=stop`) | OK |
| GET | `/import/watcher/status` | Whether the watcher is running | Data |
| GET | `/import/watcher/toggle-state` | Whether the watcher is running | Data |

---

## Jobs

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/jobs` | Jobs page | Page |
| GET | `/jobs/all` | Every job, with links to its progress and logs | JSON |
| GET | `/jobs/active` | Running and waiting jobs | Data |
| GET | `/jobs/list` | Jobs, optionally only those of one type (`?prefix=download_`) | Data |
| GET | `/jobs/latest` | The 5 most recent jobs | Data |
| GET | `/jobs/count` | Number of jobs (`?filter=active` for running and waiting ones) | Value |
| POST | `/jobs/start/:type` | Start a job of the given [type](jobs.md#job-types), e.g. `calculate_metrics` | Job |
| POST | `/jobs/clear-finished` | Remove finished jobs and their logs | OK |
| GET | `/jobs/:id` | One job's status, progress and message | JSON |
| GET | `/jobs/:id/progress` | One job's progress | Data |
| GET | `/jobs/:id/logs` | One job's log as plain text (`?color=true` for a colored page) | Text |
| POST | `/jobs/:id/cancel` | Cancel a job | Data |

---

## Downloading

Add `?downloader=<name>` to choose which [plugin](downloading.md) to use. Without it, the
first configured plugin is used.

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/downloads` | Download page | Page |
| POST | `/downloads/search` | Search (`query`, `type`: `album`, `track`, `artist` or `link`, `limit`) | Data |
| POST | `/downloads/search/albums` | Search albums (`query`) | Data |
| POST | `/downloads/search/tracks` | Search tracks (`query`) | Data |
| GET | `/downloads/album/:albumId/tracks` | An album's tracks | Data |
| GET | `/downloads/chart/tracks` | The service's top tracks | Data |
| GET | `/downloads/user/info` | Your account on the service, and its status | Data |
| GET | `/downloads/capabilities` | What the plugin supports | JSON |
| POST | `/downloads/track` | Download a track (`trackId`) | Job |
| POST | `/downloads/album` | Download an album (`albumId`) | Job |
| POST | `/downloads/artist` | Download an artist's discography (`artistId`) | Job |
| POST | `/downloads/tracks` | Download several tracks (`trackIds`, comma-separated) | Job |
| POST | `/downloads/playlist` | Download a playlist (`trackIds`, `playlistName`) | Job |

---

## Lyrics

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/analyze/lyrics` | Lyrics analysis page | Page |
| POST | `/analyze/lyrics` | Start a [lyrics analysis](features/lyrics.md#lyrics-analysis) (`provider`, `skip_existing_lyrics`, `override_no_queue`) | Job |
| GET | `/tag/:trackId/lyrics` | Enabled lyrics providers | Data |
| GET | `/tag/:trackId/lyrics/text/:provider` | Look up a track's lyrics with a provider | JSON |
| GET | `/library/tracks/:id/lyrics` | A track's lyrics | Value |
| GET | `/lyrics/queue/items` | Tracks in the lyrics queue | Data |
| GET | `/lyrics/queue/items/grouped` | Queue grouped by `?type=artist` or `?type=album` | Data |
| GET | `/lyrics/queue/header` | Queue summary | Data |
| GET | `/lyrics/queue/count` | Number of tracks in the queue | Value |
| GET | `/lyrics/queue/:id/new_lyrics` | The new lyrics found for a queued track | Value |
| POST | `/lyrics/queue/:id/:action` | Act on a queued track: `override`, `keep_old`, `no_lyrics`, `skip` | OK |
| POST | `/lyrics/queue/group/:groupType/:groupKey/:action` | Act on a whole artist or album group | OK |
| POST | `/lyrics/queue/clear` | Empty the queue | OK |

---

## Playlists

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/playlists` | Playlists page | Page |
| GET | `/playlists/:id` | One playlist and its tracks | Data |
| GET | `/playlists/create-modal` | New playlist form | Data |
| GET | `/playlists/:type/:id/playlists` | Playlists you can add a track, album or artist to | Data |
| GET | `/playlists/:id/export` | Download the playlist as `.m3u` | File |
| POST | `/playlists/` | Create a playlist (`name`, `description`) | OK |
| PUT | `/playlists/:id` | Rename or edit a playlist (`name`, `description`) | OK |
| DELETE | `/playlists/:id` | Delete a playlist | OK |
| POST | `/playlists/items` | Add music (`playlist_id`, `item_type`: `track`, `album` or `artist`, `item_id`) | OK |
| DELETE | `/playlists/:playlistId/tracks/:trackId` | Remove a track from a playlist | OK |

---

## File paths (Reorganize)

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/analyze/files` | File paths page | Page |
| POST | `/analyze/reorganize` | Start a [reorganize](features/reorganize.md) (`fat32_safe=true` to also make paths FAT32-safe) | Job |

---

## Merge

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/analyze/merge` | Merge page | Page |

Finding and merging duplicates is only available from the web interface. See
[Merge](features/merge.md).

---

## Metrics

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/metrics/overview` | Number of tracks, artists and albums | Data |
| GET | `/metrics/charts/genre` | Tracks per genre | Data |
| GET | `/metrics/charts/year` | Tracks per release year | Data |
| GET | `/metrics/charts/format` | Tracks per audio format | Data |
| GET | `/metrics/charts/metadata` | Percentage of tracks with each tag | Data |

To recalculate the metrics, start a `calculate_metrics` job with
`POST /jobs/start/calculate_metrics`.

---

## Streaming

| Method | Route | What it does | Response |
|--------|-------|--------------|----------|
| GET | `/stream?path=<url-encoded path>` | Play or download an audio file from the library or download folder | File |
