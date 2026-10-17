---
weight: 100
title: "MCP"
description: "A Model Context Protocol endpoint that lets AI agents drive Soulsolid."
icon: "smart_toy"
draft: false
toc: true
---

The **mcp** feature exposes Soulsolid's domain services to AI agents through a
[Model Context Protocol](https://modelcontextprotocol.io) endpoint (Streamable HTTP)
mounted on the main server at `POST /mcp`. It is just another transport: tools call
the same services the web UI handlers use.

## Configuration

```yaml
mcp:
  enabled: true
  allowed_origins: []         # browser origins allowed to call /mcp
  read_only: false            # expose only tools that don't modify anything
```

The endpoint is stateless (no sessions, no SSE stream); every tool call is a single
JSON request/response.

## Tools

| Area       | Tools |
|------------|-------|
| Library    | `search_library`, `get_track`, `get_album`, `get_artist`, `library_stats`, `get_library_tree`, `get_track_lyrics`, `delete_track`, `delete_album`, `delete_artist` |
| Metadata   | `update_track_tags`, `search_metadata_provider`, `get_provider_track`, `fingerprint_track`, `get_fingerprint`, `run_acoustid_analysis` |
| Lyrics     | `fetch_lyrics`, `run_lyrics_analysis`, `list_lyrics_queue`, `get_new_lyrics`, `resolve_lyrics_item`, `resolve_lyrics_group`, `clear_lyrics_queue` |
| Importing  | `import_path`, `list_import_queue`, `resolve_queue_item`, `resolve_import_group`, `clear_import_queue`, `get_watcher_status`, `toggle_watcher`, `run_reorganize` |
| Downloads  | `get_download_capabilities`, `get_downloader_info`, `search_downloads`, `download_track`, `download_album`, `download_artist`, `download_tracks`, `download_playlist` |
| Playlists  | `list_playlists`, `get_playlist`, `create_playlist`, `update_playlist`, `delete_playlist`, `add_to_playlist`, `remove_from_playlist`, `get_track_playlists` |
| Merge      | `find_merge_groups`, `merge_metadata`, `merge_tracks`, `keep_duplicate_tracks` |
| Jobs       | `list_jobs`, `get_job`, `get_job_logs`, `start_job`, `cancel_job`, `clear_finished_jobs` |
| Config     | `get_config`, `update_settings` |

Every tool carries MCP `readOnlyHint` / `destructiveHint` annotations, so clients can
ask for confirmation before deleting files, rewriting tags or changing settings.

## Security

The MCP endpoint has **no authentication**: anyone who can reach the port can use
every tool, including deleting tracks and files, rewriting tags and changing settings.
Only enable it on a trusted network, or put it behind a reverse proxy that
authenticates requests.

- **Origin check.** Requests carrying an `Origin` header are rejected unless it is
  listed in `allowed_origins`. This blocks malicious web pages, including DNS-rebinding
  attacks against a server on `localhost` or the LAN. Desktop and CLI MCP clients send
  no `Origin` and are unaffected.
- **`read_only: true`** registers only read tools. Use it for agents that should
  browse and search but never modify anything.
- **No secrets in tool output.** `get_config` masks the Telegram token,
  provider secrets, downloader plugin config and the webhook command.
- **Restricted settings.** `update_settings` cannot change `libraryPath`,
  `downloadPath`, `jobs.log_path`, the Telegram token or allowed users, or provider
  secrets. Path templates must be relative and cannot contain `..`.
- **Imports are confined to `downloadPath`.** `import_path` only accepts directories
  inside the download path (symlinks are resolved), since imports may move files.
- **Bounded output.** `get_job_logs` returns the last lines of a log, `get_library_tree`
  is truncated at 64 KiB, and search tools return at most 100 results per call.
- **Prompt injection.** Track metadata, lyrics and search results come from files and
  third-party services, and agents read them as input. Use a client that asks before
  running destructive tools, or run with `read_only: true`.
- Request bodies are capped at 1 MiB.
