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

// --- Tool registration ---

func registerTools(s *server.MCPServer) {
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

	s.AddTool(mcp.NewTool("list_jobs",
		mcp.WithDescription("List background jobs (downloads, imports, analysis). Optionally filter by status."),
		mcp.WithString("status", mcp.Description("Filter by status: pending, running, completed, failed, cancelled")),
	), listJobs)

	s.AddTool(mcp.NewTool("get_job",
		mcp.WithDescription("Get current status and progress of a background job by ID"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID")),
	), getJob)

	s.AddTool(mcp.NewTool("get_config",
		mcp.WithDescription("Get the current SoulSolid configuration"),
	), getConfig)

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

	result := make(map[string]string, len(endpoints))
	for _, e := range endpoints {
		body, err := apiGet(e.path)
		if err != nil {
			result[e.key] = "error: " + err.Error()
		} else {
			result[e.key] = strings.TrimSpace(string(body))
		}
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
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

func getConfig(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	body, err := apiGet("/config")
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

