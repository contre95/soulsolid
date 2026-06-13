# SoulSolid API Reference

Most endpoints that return Resource responses or support both HTML and API clients perform content negotiation.
HTMX sends `HX-Request: true` automatically; anything else is treated as an API client.
Resource responses additionally negotiate via the `Accept` header (`Accept: application/json` returns JSON metadata instead of the binary).
Some endpoints are always JSON or HTMX-only and do not perform `HX-Request`-based negotiation.

**Response types**

| Type | HTMX | API (no `HX-Request`) |
|------|------|-----------------------|
| **Section** | Section partial (`sections/<name>`) | Full page via `main.html` |
| **Partial** | HTML fragment | Same data as JSON |
| **Text** | Plain string | `{"key":"…","value":…}` |
| **Toast OK** | Success toast | `{"message":"…"}` |
| **Toast Err** | Error toast | `{"error":"…"}` + HTTP status |
| **Toast Job** | Success toast | `202 {"job_id":"…"}` |
| **Resource** | Binary / file (default) | `{"type":"…","url":"…"}` when `Accept: application/json` |
| **JSON** | — | Always JSON, no negotiation |

---

## UI / Dashboard

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/` | Section | `sections/dashboard` | full page |
| GET | `/dashboard` | Section | `sections/dashboard` | full page |
| GET | `/analyze` | Section | `sections/analyze` | full page |
| GET | `/dashboard/quick-actions` | Partial | HTML card | JSON data |

---

## Config

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/settings` | Section | `sections/settings` | full page |
| GET | `/config/form` | Partial | HTML form | JSON config |
| PUT | `/settings` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/config` | JSON | — | config struct as JSON |
| GET | `/config?fmt=yaml` | — | raw `text/yaml` | raw `text/yaml` |
| GET | `/config/database/download` | Resource | SQLite file download | `{"type":"application/octet-stream","url":"…"}` |

---

## Library

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/library` | Section | `sections/library` | full page |
| GET | `/library/table` | Partial | HTML table | JSON data |
| GET | `/library/tracks/:trackId/overview` | Partial | HTML panel | JSON data |
| GET | `/library/search` | Partial | HTML results list | JSON results + pagination |
| GET | `/library/artists/count` | Text | `"N"` | `{"key":"artists_count","value":N}` |
| GET | `/library/albums/count` | Text | `"N"` | `{"key":"albums_count","value":N}` |
| GET | `/library/tracks/count` | Text | `"N tracks"` | `{"key":"tracks_count","value":N}` |
| GET | `/library/storage/size` | Text | `"X GB"` | `{"key":"storage_size_bytes","value":N}` |
| GET | `/library/artists/:id` | JSON | — | artist object |
| GET | `/library/albums/:id` | JSON | — | album object |
| GET | `/library/tracks/:id` | JSON | — | track object |
| GET | `/library/tree` | Text | plain tree string | `{"key":"file_tree","value":"…"}` |
| GET | `/library/tracks/:id/lyrics` | Text | plain lyrics | `{"key":"lyrics","value":"…"}` |
| DELETE | `/library/tracks/:trackId` | Toast OK | success toast | `{"message":"…"}` |
| DELETE | `/library/albums/:albumId` | Toast OK | success toast | `{"message":"…"}` |
| DELETE | `/library/artists/:artistId` | Toast OK | success toast | `{"message":"…"}` |

---

## Tag / Metadata

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/tag/:trackId` | Section | `sections/tag` | full page |
| GET | `/tag/:trackId?source=db` | Section | `sections/tag` (reads DB) | full page |
| POST | `/tag/:trackId` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/tag/:trackId/:provider` | Section | `sections/tag` (provider data) | full page |
| GET | `/tag/:trackId/artwork` | Resource | image bytes | `{"type":"image/…","url":"…"}` |
| GET | `/tag/:trackId/fingerprint` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/tag/:trackId/fingerprint/view` | Text | fingerprint string | `{"key":"fingerprint","value":"…"}` |
| GET | `/tag/:trackId/search/:provider` | Partial | HTML modal | JSON results |
| GET | `/tag/:trackId/select/:provider` | Partial | HTML form | JSON track data |
| POST | `/analyze/acoustid` | Toast Job | success toast | `202 {"job_id":"…"}` |
| GET | `/analyze/metadata` | Section | `sections/analyze_metadata` | full page |

---

## Importing

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/import` | Section | `sections/import` | full page |
| GET | `/import/directory/form` | Partial | HTML form | JSON data |
| GET | `/import/queue/items` | Partial | HTML list | JSON items |
| GET | `/import/queue/items/grouped` | Partial | HTML grouped list | JSON groups |
| GET | `/import/queue/header` | Partial | HTML header | JSON data |
| GET | `/import/queue/:id/artwork` | Resource | image bytes | `{"type":"image/…","url":"…"}` |
| GET | `/import/queue/count` | Text | `"(N)"` or `""` | `{"key":"queue_count","value":N}` |
| POST | `/import/directory` | Toast Job | success toast | `202 {"job_id":"…"}` |
| POST | `/import/queue/:id/:action` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/import/queue/group/:groupType/:groupKey/:action` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/import/queue/clear` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/import/prune/download-path` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/import/watcher/toggle` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/import/watcher/status` | Partial | HTML status | JSON status |
| GET | `/import/watcher/toggle-state` | Partial | HTML toggle | JSON state |

---

## Jobs

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/jobs` | Section | `sections/jobs` | full page |
| GET | `/jobs/active` | Partial | HTML active list | JSON jobs |
| GET | `/jobs/list` | Partial | HTML list | JSON jobs |
| GET | `/jobs/latest` | Partial | HTML latest list | JSON jobs |
| GET | `/jobs/count` | Text | `"(N)"` or `""` | `{"key":"jobs_count","value":N}` |
| POST | `/jobs/clear-finished` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/jobs/all` | JSON | — | `[{job, _links}]` |
| POST | `/jobs/start/:type` | Toast Job | success toast | `202 {"job_id":"…"}` |
| GET | `/jobs/:id` | JSON | — | `{job, _links}` |
| GET | `/jobs/:id/progress` | Partial | HTML progress bar | JSON progress |
| GET | `/jobs/:id/logs` | — | plain text | plain text |
| GET | `/jobs/:id/logs?color=true` | — | colored HTML fragment | fullscreen HTML page |
| POST | `/jobs/:id/cancel` | Partial | HTML job card | JSON job data |

---

## Downloading

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/downloads` | Section | `sections/download` | full page |
| GET | `/downloads/chart/tracks` | Partial | HTML chart | JSON tracks |
| POST | `/downloads/search` | Partial | HTML results | JSON results |
| POST | `/downloads/search/albums` | Partial | HTML results | JSON albums |
| POST | `/downloads/search/tracks` | Partial | HTML results | JSON tracks |
| GET | `/downloads/album/:albumId/tracks` | Partial | HTML track list | JSON tracks |
| GET | `/downloads/user/info` | Partial | HTML user info | JSON user info |
| GET | `/downloads/capabilities` | JSON | — | capabilities object |
| POST | `/downloads/track` | Toast Job | success toast | `202 {"job_id":"…"}` |
| POST | `/downloads/album` | Toast Job | success toast | `202 {"job_id":"…"}` |
| POST | `/downloads/artist` | Toast Job | success toast | `202 {"job_id":"…"}` |
| POST | `/downloads/tracks` | Toast Job | success toast | `202 {"job_id":"…"}` |
| POST | `/downloads/playlist` | Toast Job | success toast | `202 {"job_id":"…"}` |

---

## Lyrics

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/analyze/lyrics` | Section | `sections/analyze_lyrics` | full page |
| GET | `/tag/:trackId/lyrics/text/:provider` | — | plain lyrics text | `{"track_id":"…","lyrics":"…"}` |
| GET | `/library/tracks/:id/lyrics` | Text | plain lyrics | `{"key":"lyrics","value":"…"}` |
| GET | `/lyrics/queue/header` | Partial | HTML header | JSON data |
| GET | `/lyrics/queue/items` | Partial | HTML list | JSON items |
| GET | `/lyrics/queue/items/grouped` | Partial | HTML grouped list | JSON groups |
| GET | `/lyrics/queue/count` | Text | `"(N)"` or `""` | `{"key":"queue_count","value":N}` |
| GET | `/lyrics/queue/:id/new_lyrics` | Text | plain lyrics | `{"key":"lyrics","value":"…"}` |
| POST | `/lyrics/queue/:id/:action` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/lyrics/queue/group/:groupType/:groupKey/:action` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/lyrics/queue/clear` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/analyze/lyrics` | Toast Job | success toast | `202 {"job_id":"…"}` |

---

## Playlists

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/playlists` | Section | `sections/playlists` | full page |
| GET | `/playlists/:id` | Partial | HTML playlist view | JSON playlist |
| GET | `/playlists/create-modal` | Partial | HTML modal | JSON data |
| GET | `/playlists/:type/:id/playlists` | Partial | HTML list | JSON playlists |
| GET | `/playlists/:id/export` | Resource | `.m3u` file | `{"type":"audio/x-mpegurl","url":"…"}` |
| POST | `/playlists/` | Toast OK | success toast | `{"message":"…"}` |
| PUT | `/playlists/:id` | Toast OK | success toast | `{"message":"…"}` |
| DELETE | `/playlists/:id` | Toast OK | success toast | `{"message":"…"}` |
| POST | `/playlists/items` | Toast OK | success toast | `{"message":"…"}` |
| DELETE | `/playlists/:playlistId/tracks/:trackId` | Toast OK | success toast | `{"message":"…"}` |

---

## Reorganize

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/analyze/files` | Section | `sections/analyze_files` | full page |
| POST | `/analyze/reorganize` | Toast Job | success toast | `202 {"job_id":"…"}` |

---

## Metrics

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/metrics/overview` | Partial | HTML overview | JSON metrics |
| GET | `/metrics/charts/genre` | Partial | HTML chart | JSON data |
| GET | `/metrics/charts/year` | Partial | HTML chart | JSON data |
| GET | `/metrics/charts/format` | Partial | HTML chart | JSON data |
| GET | `/metrics/charts/metadata` | Partial | HTML chart | JSON data |

---

## Streaming

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/stream?path=<encoded-path>` | Resource | audio bytes | `{"type":"audio/…","url":"…"}` when `Accept: application/json` |
