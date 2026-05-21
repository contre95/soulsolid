# SoulSolid API Reference

Every endpoint supports dual content negotiation via the `HX-Request` header.  
HTMX sends `HX-Request: true` automatically; anything else is treated as an API client.

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
| GET | `/` | — | redirect → `/ui` | redirect → `/ui` |
| GET | `/ui` | Section | `sections/dashboard` | full page |
| GET | `/ui/dashboard` | Section | `sections/dashboard` | full page |
| GET | `/ui/analyze` | Section | `sections/analyze` | full page |
| GET | `/ui/quick-actions-card` | Partial | HTML card | JSON data |

---

## Config

| Method | Route | Type | HTMX | API / Browser |
|--------|-------|------|------|---------------|
| GET | `/ui/settings` | Section | `sections/settings` | full page |
| GET | `/ui/config/form` | Partial | HTML form | JSON config |
| POST | `/settings/update` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/config` | JSON | — | config struct as JSON |
| GET | `/config?fmt=yaml` | — | raw `text/yaml` | raw `text/yaml` |
| GET | `/config/database/download` | Resource | SQLite file download | `{"type":"application/octet-stream","url":"…"}` |

---

## Library

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/ui/library` | Section | `sections/library` | full page |
| GET | `/ui/library/table` | Partial | HTML table | JSON data |
| GET | `/ui/library/tracks/:trackId/overview` | Partial | HTML panel | JSON data |
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
| GET | `/tag/buttons/metadata/:trackId` | Partial | HTML buttons | JSON data |
| POST | `/analyze/acoustid` | Toast Job | success toast | `202 {"job_id":"…"}` |
| GET | `/ui/analyze/metadata` | Section | `sections/analyze_metadata` | full page |

---

## Importing

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/ui/import` | Section | `sections/import` | full page |
| GET | `/ui/importing/directory/form` | Partial | HTML form | JSON data |
| GET | `/ui/importing/queue/items` | Partial | HTML list | JSON items |
| GET | `/ui/importing/queue/items/grouped` | Partial | HTML grouped list | JSON groups |
| GET | `/ui/importing/queue/header` | Partial | HTML header | JSON data |
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
| GET | `/ui/jobs` | Section | `sections/jobs` | full page |
| GET | `/ui/jobs/active` | Partial | HTML active list | JSON jobs |
| GET | `/ui/jobs/list` | Partial | HTML list | JSON jobs |
| GET | `/ui/jobs/latest` | Partial | HTML latest list | JSON jobs |
| GET | `/ui/jobs/count` | Text | `"(N)"` or `""` | `{"key":"jobs_count","value":N}` |
| POST | `/ui/jobs/clear-finished` | Toast OK | success toast | `{"message":"…"}` |
| GET | `/jobs/` | JSON | — | `[{job, _links}]` |
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
| GET | `/ui/download` | Section | `sections/download` | full page |
| GET | `/ui/downloading/chart/tracks` | Partial | HTML chart | JSON tracks |
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
| GET | `/ui/lyrics/queue/header` | Partial | HTML header | JSON data |
| GET | `/ui/lyrics/queue/items` | Partial | HTML list | JSON items |
| GET | `/ui/lyrics/queue/items/grouped` | Partial | HTML grouped list | JSON groups |
| GET | `/ui/analyze/lyrics` | Section | `sections/analyze_lyrics` | full page |
| GET | `/tag/buttons/lyrics/:trackId` | Partial | HTML buttons | JSON data |
| GET | `/tag/:trackId/lyrics/text/:provider` | — | plain lyrics text | `{"track_id":"…","lyrics":"…"}` |
| GET | `/library/tracks/:id/lyrics` | Text | plain lyrics | `{"key":"lyrics","value":"…"}` |
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
| GET | `/ui/playlists` | Section | `sections/playlists` | full page |
| GET | `/ui/playlists/:id` | Partial | HTML playlist view | JSON playlist |
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
| GET | `/ui/analyze/files` | Section | `sections/analyze_files` | full page |
| POST | `/analyze/reorganize` | Toast Job | success toast | `202 {"job_id":"…"}` |

---

## Metrics

| Method | Route | Type | HTMX | API |
|--------|-------|------|------|-----|
| GET | `/ui/metrics/overview` | Partial | HTML overview | JSON metrics |
| GET | `/ui/metrics/charts/genre` | Partial | HTML chart | JSON data |
| GET | `/ui/metrics/charts/year` | Partial | HTML chart | JSON data |
| GET | `/ui/metrics/charts/format` | Partial | HTML chart | JSON data |
| GET | `/ui/metrics/charts/metadata` | Partial | HTML chart | JSON data |
