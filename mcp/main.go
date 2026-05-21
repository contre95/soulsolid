package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	baseURL    string
	httpClient = &http.Client{}
)

func main() {
	flag.StringVar(&baseURL, "base-url", "http://localhost:8080", "SoulSolid base URL")
	flag.Parse()
	baseURL = strings.TrimRight(baseURL, "/")

	s := server.NewMCPServer("soulsolid", "1.0.0")
	registerTools(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

// --- HTTP helpers ---

func apiGet(path string) ([]byte, error) {
	resp, err := httpClient.Get(baseURL + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func apiPostForm(path string, data url.Values) ([]byte, int, error) {
	resp, err := httpClient.PostForm(baseURL+path, data)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func apiPostJSON(path string, body any) ([]byte, int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	resp, err := httpClient.Post(baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func apiDelete(path string) (int, error) {
	req, _ := http.NewRequest(http.MethodDelete, baseURL+path, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func apiPutForm(path string, data url.Values) ([]byte, int, error) {
	req, _ := http.NewRequest(http.MethodPut, baseURL+path, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func apiPostEmpty(path string) ([]byte, int, error) {
	req, _ := http.NewRequest(http.MethodPost, baseURL+path, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// --- Result helpers ---

func jsonResult(data []byte) *mcp.CallToolResult {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return mcp.NewToolResultText(string(data))
	}
	pretty, _ := json.MarshalIndent(v, "", "  ")
	return mcp.NewToolResultText(string(pretty))
}

func errResult(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(msg)
}

func strArg(args any, key string) string {
	m, ok := args.(map[string]any)
	if !ok {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func intArg(args any, key string, def int) int {
	m, ok := args.(map[string]any)
	if !ok {
		return def
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return def
}

func boolArg(args any, key string) bool {
	m, ok := args.(map[string]any)
	if !ok {
		return false
	}
	v, _ := m[key].(bool)
	return v
}

// --- Tool registration ---

func registerTools(s *server.MCPServer) {
	// Library
	s.AddTool(mcp.NewTool("search_library",
		mcp.WithDescription("Search the SoulSolid music library. Returns tracks, albums, and artists. Use the 'ID' and 'type' from results with get_track / get_album / get_artist."),
		mcp.WithString("query", mcp.Description("Search terms — matches title, artist name, album title")),
		mcp.WithNumber("limit", mcp.Description("Max results per page (default 20)")),
		mcp.WithNumber("page", mcp.Description("Page number, 1-indexed (default 1)")),
	), searchLibrary)

	s.AddTool(mcp.NewTool("get_track",
		mcp.WithDescription("Get full metadata for a track by its UUID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Track UUID")),
	), getTrack)

	s.AddTool(mcp.NewTool("get_artist",
		mcp.WithDescription("Get artist details by UUID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Artist UUID")),
	), getArtist)

	s.AddTool(mcp.NewTool("get_album",
		mcp.WithDescription("Get album details by UUID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Album UUID")),
	), getAlbum)

	s.AddTool(mcp.NewTool("library_stats",
		mcp.WithDescription("Get library overview: track count, artist count, album count, total storage size"),
	), libraryStats)

	s.AddTool(mcp.NewTool("get_library_tree",
		mcp.WithDescription("Get the full file-system tree of the music library as a plain text string"),
	), getLibraryTree)

	// Tag / Metadata
	s.AddTool(mcp.NewTool("update_track_tags",
		mcp.WithDescription("Save tag metadata to a track's file. Only provided fields are updated; omit a field to leave it unchanged."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
		mcp.WithString("title", mcp.Description("Track title")),
		mcp.WithString("artist", mcp.Description("Artist name")),
		mcp.WithString("album", mcp.Description("Album title")),
		mcp.WithString("album_artist", mcp.Description("Album artist")),
		mcp.WithString("year", mcp.Description("Release year")),
		mcp.WithString("genre", mcp.Description("Genre")),
		mcp.WithString("track_number", mcp.Description("Track number (e.g. '1' or '1/12')")),
		mcp.WithString("disc_number", mcp.Description("Disc number")),
		mcp.WithString("comment", mcp.Description("Comment field")),
		mcp.WithString("lyrics", mcp.Description("Embedded lyrics text")),
	), updateTrackTags)

	s.AddTool(mcp.NewTool("search_metadata_provider",
		mcp.WithDescription("Search an external metadata provider (e.g. musicbrainz) for tag data matching a track. Returns candidate results to pick from."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
		mcp.WithString("provider", mcp.Required(), mcp.Description("Metadata provider name (e.g. 'musicbrainz')")),
	), searchMetadataProvider)

	s.AddTool(mcp.NewTool("get_provider_track",
		mcp.WithDescription("Preview the tag data a provider would write for a specific search result. Use after search_metadata_provider to inspect a candidate before saving with update_track_tags."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
		mcp.WithString("provider", mcp.Required(), mcp.Description("Metadata provider name")),
	), getProviderTrack)

	s.AddTool(mcp.NewTool("fingerprint_track",
		mcp.WithDescription("Generate an acoustic fingerprint for a track. The fingerprint is used by AcoustID for automatic metadata identification."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
	), fingerprintTrack)

	s.AddTool(mcp.NewTool("get_fingerprint",
		mcp.WithDescription("Get the stored acoustic fingerprint string for a track"),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
	), getFingerprint)

	s.AddTool(mcp.NewTool("run_acoustid_analysis",
		mcp.WithDescription("Start a background job that fingerprints all unidentified tracks and looks them up via AcoustID. Returns a job ID."),
	), runAcoustidAnalysis)

	s.AddTool(mcp.NewTool("run_lyrics_analysis",
		mcp.WithDescription("Start a background job that fetches and embeds lyrics for all tracks missing them. Returns a job ID."),
	), runLyricsAnalysis)

	s.AddTool(mcp.NewTool("run_reorganize",
		mcp.WithDescription("Start a background job that reorganizes music files according to the configured naming template. Returns a job ID."),
	), runReorganize)

	// Jobs
	s.AddTool(mcp.NewTool("list_jobs",
		mcp.WithDescription("List background jobs (downloads, imports, analysis). Optionally filter by status."),
		mcp.WithString("status", mcp.Description("Filter by status: pending, running, completed, failed, cancelled")),
	), listJobs)

	s.AddTool(mcp.NewTool("get_job",
		mcp.WithDescription("Get current status and progress of a background job by ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID")),
	), getJob)

	s.AddTool(mcp.NewTool("get_job_logs",
		mcp.WithDescription("Get the plain-text log output for a background job"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID")),
	), getJobLogs)

	s.AddTool(mcp.NewTool("start_job",
		mcp.WithDescription("Start a named background job type. Use list_jobs to see existing jobs; use this for one-off triggers like 'scan' or 'cleanup'."),
		mcp.WithString("type", mcp.Required(), mcp.Description("Job type name (e.g. 'scan', 'cleanup')")),
	), startJob)

	s.AddTool(mcp.NewTool("cancel_job",
		mcp.WithDescription("Cancel a running or pending background job"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID")),
	), cancelJob)

	s.AddTool(mcp.NewTool("clear_finished_jobs",
		mcp.WithDescription("Remove all completed and failed jobs from the job history"),
	), clearFinishedJobs)

	// Config
	s.AddTool(mcp.NewTool("get_config",
		mcp.WithDescription("Get the current SoulSolid configuration"),
	), getConfig)

	s.AddTool(mcp.NewTool("update_settings",
		mcp.WithDescription("Update one or more SoulSolid settings. Pass key=value pairs as a JSON object in the 'settings' field."),
		mcp.WithString("settings", mcp.Required(), mcp.Description("JSON object of settings key/value pairs to update, e.g. {\"music_dir\":\"/music\"}")),
	), updateSettings)

	// Importing
	s.AddTool(mcp.NewTool("list_import_queue",
		mcp.WithDescription("List all items currently in the import queue (tracks awaiting review before being added to the library)"),
	), listImportQueue)

	s.AddTool(mcp.NewTool("resolve_queue_item",
		mcp.WithDescription("Resolve an import queue item. Actions: 'import' (add to library), 'replace' (replace existing), 'delete' (remove the file), 'skip' (dismiss)."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Queue item ID (from list_import_queue)")),
		mcp.WithString("action", mcp.Required(), mcp.Description("One of: import, replace, delete, skip")),
	), resolveQueueItem)

	s.AddTool(mcp.NewTool("resolve_import_group",
		mcp.WithDescription("Apply an action to a whole group of import queue items at once (e.g. all tracks from the same album)."),
		mcp.WithString("group_type", mcp.Required(), mcp.Description("Group type (e.g. 'album', 'artist')")),
		mcp.WithString("group_key", mcp.Required(), mcp.Description("Group key value")),
		mcp.WithString("action", mcp.Required(), mcp.Description("One of: import, replace, delete, skip")),
	), resolveImportGroup)

	s.AddTool(mcp.NewTool("clear_import_queue",
		mcp.WithDescription("Remove all items from the import queue without importing them"),
	), clearImportQueue)

	s.AddTool(mcp.NewTool("import_path",
		mcp.WithDescription("Trigger an import job for a directory on the server filesystem. Use list_jobs / get_job to track progress."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute directory path on the server to import")),
	), importPath)

	s.AddTool(mcp.NewTool("get_watcher_status",
		mcp.WithDescription("Get the current status of the filesystem watcher (watches for new music files)"),
	), getWatcherStatus)

	s.AddTool(mcp.NewTool("toggle_watcher",
		mcp.WithDescription("Enable or disable the filesystem watcher that auto-imports new music files"),
	), toggleWatcher)

	// Downloads
	s.AddTool(mcp.NewTool("get_download_capabilities",
		mcp.WithDescription("Get the capabilities and configuration of all available downloader plugins"),
	), getDownloadCapabilities)

	s.AddTool(mcp.NewTool("get_downloader_info",
		mcp.WithDescription("Get user info and status for a downloader plugin (e.g. account details, quota). Also lists all available downloaders when called without arguments."),
		mcp.WithString("downloader", mcp.Description("Downloader plugin name (e.g. 'deezer'). Omit to list all.")),
	), getDownloaderInfo)

	s.AddTool(mcp.NewTool("search_downloads",
		mcp.WithDescription("Search an external source (e.g. Deezer) for tracks, albums, or artists to download. Returns external IDs needed for download_track / download_album / download_artist."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search terms")),
		mcp.WithString("type", mcp.Required(), mcp.Description("One of: track, album, artist, link (for direct URLs)")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
	), searchDownloads)

	s.AddTool(mcp.NewTool("download_track",
		mcp.WithDescription("Download a single track by its external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("External track ID (from search_downloads results)")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
	), downloadTrack)

	s.AddTool(mcp.NewTool("download_album",
		mcp.WithDescription("Download a full album by its external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcp.WithString("album_id", mcp.Required(), mcp.Description("External album ID (from search_downloads results)")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
	), downloadAlbum)

	s.AddTool(mcp.NewTool("download_artist",
		mcp.WithDescription("Download all albums from an artist by their external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcp.WithString("artist_id", mcp.Required(), mcp.Description("External artist ID (from search_downloads results)")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
	), downloadArtist)

	s.AddTool(mcp.NewTool("download_tracks",
		mcp.WithDescription("Download multiple tracks at once by their external IDs. Returns a job ID. Pass IDs as a comma-separated string."),
		mcp.WithString("track_ids", mcp.Required(), mcp.Description("Comma-separated list of external track IDs")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
	), downloadTracks)

	s.AddTool(mcp.NewTool("download_playlist",
		mcp.WithDescription("Download all tracks from an external playlist by URL or ID. Returns a job ID."),
		mcp.WithString("playlist_id", mcp.Required(), mcp.Description("External playlist ID or URL")),
		mcp.WithString("downloader", mcp.Required(), mcp.Description("Downloader plugin name (e.g. 'deezer')")),
	), downloadPlaylist)

	// Lyrics
	s.AddTool(mcp.NewTool("get_track_lyrics",
		mcp.WithDescription("Get the lyrics currently embedded in a track's metadata"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Track UUID")),
	), getTrackLyrics)

	s.AddTool(mcp.NewTool("fetch_lyrics",
		mcp.WithDescription("Fetch lyrics for a track from an external provider (e.g. lrclib). Returns the lyrics text without saving — use to preview before embedding."),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID")),
		mcp.WithString("provider", mcp.Required(), mcp.Description("Lyrics provider name (e.g. 'lrclib')")),
	), fetchLyrics)

	s.AddTool(mcp.NewTool("list_lyrics_queue",
		mcp.WithDescription("List all tracks in the lyrics review queue (tracks where fetched lyrics need approval before saving)"),
	), listLyricsQueue)

	s.AddTool(mcp.NewTool("get_new_lyrics",
		mcp.WithDescription("Preview the fetched lyrics for a lyrics queue item before accepting or rejecting it"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Lyrics queue item ID (from list_lyrics_queue)")),
	), getNewLyrics)

	s.AddTool(mcp.NewTool("resolve_lyrics_item",
		mcp.WithDescription("Accept or reject a lyrics queue item. Actions: 'accept' (embed lyrics), 'reject' (discard), 'skip'."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Lyrics queue item ID")),
		mcp.WithString("action", mcp.Required(), mcp.Description("One of: accept, reject, skip")),
	), resolveLyricsItem)

	s.AddTool(mcp.NewTool("resolve_lyrics_group",
		mcp.WithDescription("Apply an action to a whole group of lyrics queue items at once."),
		mcp.WithString("group_type", mcp.Required(), mcp.Description("Group type (e.g. 'album', 'artist')")),
		mcp.WithString("group_key", mcp.Required(), mcp.Description("Group key value")),
		mcp.WithString("action", mcp.Required(), mcp.Description("One of: accept, reject, skip")),
	), resolveLyricsGroup)

	s.AddTool(mcp.NewTool("clear_lyrics_queue",
		mcp.WithDescription("Discard all items from the lyrics review queue"),
	), clearLyricsQueue)

	// Playlists
	s.AddTool(mcp.NewTool("create_playlist",
		mcp.WithDescription("Create a new empty playlist"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Playlist name")),
		mcp.WithString("description", mcp.Description("Optional description")),
	), createPlaylist)

	s.AddTool(mcp.NewTool("update_playlist",
		mcp.WithDescription("Rename or update the description of a playlist"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Playlist UUID")),
		mcp.WithString("name", mcp.Required(), mcp.Description("New playlist name")),
		mcp.WithString("description", mcp.Description("New description")),
	), updatePlaylist)

	s.AddTool(mcp.NewTool("delete_playlist",
		mcp.WithDescription("Permanently delete a playlist (does not delete the tracks themselves)"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Playlist UUID")),
	), deletePlaylist)

	s.AddTool(mcp.NewTool("add_to_playlist",
		mcp.WithDescription("Add a track, album, or artist to a playlist. Adding an album or artist adds all their tracks."),
		mcp.WithString("playlist_id", mcp.Required(), mcp.Description("Playlist UUID")),
		mcp.WithString("item_type", mcp.Required(), mcp.Description("One of: track, album, artist")),
		mcp.WithString("item_id", mcp.Required(), mcp.Description("UUID of the track, album, or artist to add")),
	), addToPlaylist)

	s.AddTool(mcp.NewTool("remove_from_playlist",
		mcp.WithDescription("Remove a specific track from a playlist"),
		mcp.WithString("playlist_id", mcp.Required(), mcp.Description("Playlist UUID")),
		mcp.WithString("track_id", mcp.Required(), mcp.Description("Track UUID to remove")),
	), removeFromPlaylist)

	s.AddTool(mcp.NewTool("get_item_playlists",
		mcp.WithDescription("List all playlists that contain a given track, album, or artist"),
		mcp.WithString("type", mcp.Required(), mcp.Description("One of: track, album, artist")),
		mcp.WithString("id", mcp.Required(), mcp.Description("UUID of the track, album, or artist")),
	), getItemPlaylists)

	// Deletions
	s.AddTool(mcp.NewTool("delete_track",
		mcp.WithDescription("Permanently delete a track from the library and remove its file from disk"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Track UUID")),
	), deleteTrack)

	s.AddTool(mcp.NewTool("delete_album",
		mcp.WithDescription("Permanently delete an album and all its tracks from the library"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Album UUID")),
	), deleteAlbum)

	s.AddTool(mcp.NewTool("delete_artist",
		mcp.WithDescription("Permanently delete an artist and all their tracks from the library"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Artist UUID")),
	), deleteArtist)
}

// --- Handlers ---

func searchLibrary(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	params := url.Values{}
	if q := strArg(args, "query"); q != "" {
		params.Set("query", q)
	}
	params.Set("limit", strconv.Itoa(intArg(args, "limit", 20)))
	params.Set("page", strconv.Itoa(intArg(args, "page", 1)))

	body, err := apiGet("/library/search?" + params.Encode())
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getTrack(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/library/tracks/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getArtist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/library/artists/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getAlbum(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/library/albums/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func libraryStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	type stat struct{ key, path string }
	endpoints := []stat{
		{"tracks", "/library/tracks/count"},
		{"artists", "/library/artists/count"},
		{"albums", "/library/albums/count"},
		{"storage", "/library/storage/size"},
	}

	result := make(map[string]any, len(endpoints))
	for _, e := range endpoints {
		body, err := apiGet(e.path)
		if err != nil {
			result[e.key] = "error: " + err.Error()
			continue
		}
		var parsed struct {
			Value any `json:"value"`
		}
		if err := json.Unmarshal(body, &parsed); err == nil {
			result[e.key] = parsed.Value
		} else {
			result[e.key] = strings.TrimSpace(string(body))
		}
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func getLibraryTree(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/library/tree")
	if err != nil {
		return errResult(err.Error()), nil
	}
	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		return mcp.NewToolResultText(parsed.Value), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func updateTrackTags(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	trackID := strArg(args, "track_id")

	fields := map[string]string{
		"title":        strArg(args, "title"),
		"artist":       strArg(args, "artist"),
		"album":        strArg(args, "album"),
		"album_artist": strArg(args, "album_artist"),
		"year":         strArg(args, "year"),
		"genre":        strArg(args, "genre"),
		"track_number": strArg(args, "track_number"),
		"disc_number":  strArg(args, "disc_number"),
		"comment":      strArg(args, "comment"),
		"lyrics":       strArg(args, "lyrics"),
	}

	data := url.Values{}
	for k, v := range fields {
		if v != "" {
			data.Set(k, v)
		}
	}

	body, status, err := apiPostForm("/tag/"+url.PathEscape(trackID), data)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("tags updated successfully"), nil
}

func searchMetadataProvider(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/tag/%s/search/%s",
		url.PathEscape(strArg(args, "track_id")),
		url.PathEscape(strArg(args, "provider")))
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getProviderTrack(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/tag/%s/select/%s",
		url.PathEscape(strArg(args, "track_id")),
		url.PathEscape(strArg(args, "provider")))
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func fingerprintTrack(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/tag/" + url.PathEscape(strArg(req.Params.Arguments, "track_id")) + "/fingerprint"
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getFingerprint(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/tag/" + url.PathEscape(strArg(req.Params.Arguments, "track_id")) + "/fingerprint/view"
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		return mcp.NewToolResultText(parsed.Value), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func runAcoustidAnalysis(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/analyze/acoustid")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func runLyricsAnalysis(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/analyze/lyrics")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func runReorganize(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/analyze/reorganize")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func listJobs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/jobs/"
	if s := strArg(req.Params.Arguments, "status"); s != "" {
		path += "?status=" + url.QueryEscape(s)
	}
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getJob(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/jobs/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getJobLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/jobs/" + strArg(req.Params.Arguments, "id") + "/logs")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func startJob(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/jobs/start/" + url.PathEscape(strArg(req.Params.Arguments, "type"))
	body, status, err := apiPostEmpty(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func cancelJob(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	_, status, err := apiPostJSON("/jobs/"+strArg(req.Params.Arguments, "id")+"/cancel", map[string]any{})
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("job cancelled"), nil
}

func clearFinishedJobs(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/ui/jobs/clear-finished")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("finished jobs cleared"), nil
}

func getConfig(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/config")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func updateSettings(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	raw := strArg(req.Params.Arguments, "settings")
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return errResult("settings must be a JSON object of string key/value pairs: " + err.Error()), nil
	}
	data := url.Values{}
	for k, v := range fields {
		data.Set(k, v)
	}
	body, status, err := apiPostForm("/settings/update", data)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("settings updated successfully"), nil
}

func listImportQueue(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/ui/importing/queue/items")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func resolveQueueItem(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/import/queue/%s/%s", strArg(args, "id"), strArg(args, "action"))
	_, status, err := apiPostEmpty(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("queue item resolved"), nil
}

func resolveImportGroup(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/import/queue/group/%s/%s/%s",
		url.PathEscape(strArg(args, "group_type")),
		url.PathEscape(strArg(args, "group_key")),
		strArg(args, "action"))
	body, status, err := apiPostEmpty(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("group resolved"), nil
}

func clearImportQueue(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/import/queue/clear")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("import queue cleared"), nil
}

func importPath(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	payload := map[string]any{"directoryPath": strArg(req.Params.Arguments, "path")}
	body, status, err := apiPostJSON("/import/directory", payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func getWatcherStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/import/watcher/status")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func toggleWatcher(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/import/watcher/toggle")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("watcher toggled"), nil
}

func getDownloadCapabilities(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/downloads/capabilities")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getDownloaderInfo(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/downloads/user/info"
	if d := strArg(req.Params.Arguments, "downloader"); d != "" {
		path += "?downloader=" + url.QueryEscape(d)
	}
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func searchDownloads(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	payload := map[string]any{
		"query":      strArg(args, "query"),
		"type":       strArg(args, "type"),
		"downloader": strArg(args, "downloader"),
		"limit":      intArg(args, "limit", 20),
	}
	body, status, err := apiPostJSON("/downloads/search", payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func downloadTrack(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	downloader := strArg(args, "downloader")
	payload := map[string]any{"trackId": strArg(args, "track_id")}
	body, status, err := apiPostJSON("/downloads/track?downloader="+url.QueryEscape(downloader), payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func downloadAlbum(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	downloader := strArg(args, "downloader")
	payload := map[string]any{"albumId": strArg(args, "album_id")}
	body, status, err := apiPostJSON("/downloads/album?downloader="+url.QueryEscape(downloader), payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func downloadArtist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	downloader := strArg(args, "downloader")
	payload := map[string]any{"artistId": strArg(args, "artist_id")}
	body, status, err := apiPostJSON("/downloads/artist?downloader="+url.QueryEscape(downloader), payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func downloadTracks(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	downloader := strArg(args, "downloader")
	ids := strings.Split(strArg(args, "track_ids"), ",")
	trimmed := make([]string, 0, len(ids))
	for _, id := range ids {
		if s := strings.TrimSpace(id); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	payload := map[string]any{"trackIds": trimmed}
	body, status, err := apiPostJSON("/downloads/tracks?downloader="+url.QueryEscape(downloader), payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func downloadPlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	downloader := strArg(args, "downloader")
	payload := map[string]any{"playlistId": strArg(args, "playlist_id")}
	body, status, err := apiPostJSON("/downloads/playlist?downloader="+url.QueryEscape(downloader), payload)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return jsonResult(body), nil
}

func getTrackLyrics(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/library/tracks/" + strArg(req.Params.Arguments, "id") + "/lyrics")
	if err != nil {
		return errResult(err.Error()), nil
	}
	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		return mcp.NewToolResultText(parsed.Value), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func fetchLyrics(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/tag/%s/lyrics/text/%s",
		url.PathEscape(strArg(args, "track_id")),
		url.PathEscape(strArg(args, "provider")))
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	var parsed struct {
		Lyrics string `json:"lyrics"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Lyrics != "" {
		return mcp.NewToolResultText(parsed.Lyrics), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func listLyricsQueue(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/lyrics/queue/items")
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func getNewLyrics(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := "/lyrics/queue/" + url.PathEscape(strArg(req.Params.Arguments, "id")) + "/new_lyrics"
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		return mcp.NewToolResultText(parsed.Value), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func resolveLyricsItem(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/lyrics/queue/%s/%s", url.PathEscape(strArg(args, "id")), strArg(args, "action"))
	body, status, err := apiPostEmpty(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("lyrics item resolved"), nil
}

func resolveLyricsGroup(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/lyrics/queue/group/%s/%s/%s",
		url.PathEscape(strArg(args, "group_type")),
		url.PathEscape(strArg(args, "group_key")),
		strArg(args, "action"))
	body, status, err := apiPostEmpty(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("lyrics group resolved"), nil
}

func clearLyricsQueue(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, status, err := apiPostEmpty("/lyrics/queue/clear")
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("lyrics queue cleared"), nil
}

func addToPlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	data := url.Values{
		"playlist_id": {strArg(args, "playlist_id")},
		"item_type":   {strArg(args, "item_type")},
		"item_id":     {strArg(args, "item_id")},
	}
	body, status, err := apiPostForm("/playlists/items", data)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("added successfully"), nil
}

func removeFromPlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/playlists/%s/tracks/%s", strArg(args, "playlist_id"), strArg(args, "track_id"))
	status, err := apiDelete(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("removed successfully"), nil
}

func createPlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	data := url.Values{
		"name":        {strArg(args, "name")},
		"description": {strArg(args, "description")},
	}
	body, status, err := apiPostForm("/playlists/", data)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("playlist created successfully"), nil
}

func updatePlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	data := url.Values{
		"name":        {strArg(args, "name")},
		"description": {strArg(args, "description")},
	}
	body, status, err := apiPutForm("/playlists/"+strArg(args, "id"), data)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(strings.TrimSpace(string(body))), nil
	}
	return mcp.NewToolResultText("playlist updated successfully"), nil
}

func deletePlaylist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := apiDelete("/playlists/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("playlist deleted successfully"), nil
}

func getItemPlaylists(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.Params.Arguments
	path := fmt.Sprintf("/playlists/%s/%s/playlists",
		url.PathEscape(strArg(args, "type")),
		url.PathEscape(strArg(args, "id")))
	body, err := apiGet(path)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(body), nil
}

func deleteTrack(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := apiDelete("/library/tracks/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("track deleted successfully"), nil
}

func deleteAlbum(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := apiDelete("/library/albums/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("album deleted successfully"), nil
}

func deleteArtist(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := apiDelete("/library/artists/" + strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if status >= 400 {
		return errResult(fmt.Sprintf("HTTP %d", status)), nil
	}
	return mcp.NewToolResultText("artist deleted successfully"), nil
}
