package mcp

import (
	"context"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerDownloadTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("get_download_capabilities",
		mcpgo.WithDescription("Get the status and capabilities of all available downloader plugins"),
	), s.getDownloadCapabilities)

	srv.AddTool(mcpgo.NewTool("get_downloader_info",
		mcpgo.WithDescription("Get user account info for a downloader plugin. Omit the downloader to list all available downloaders and their statuses."),
		mcpgo.WithString("downloader", mcpgo.Description("Downloader plugin name (e.g. 'deezer'). Omit to list all.")),
	), s.getDownloaderInfo)

	srv.AddTool(mcpgo.NewTool("search_downloads",
		mcpgo.WithDescription("Search an external source for tracks, albums, or artists to download. Returns external IDs needed for download_track / download_album / download_artist. Use type 'link' to resolve a direct URL."),
		mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("Search terms, or a URL when type is 'link'")),
		mcpgo.WithString("type", mcpgo.Required(), mcpgo.Description("One of: track, album, artist, link")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
		mcpgo.WithNumber("limit", mcpgo.Description("Max results (default 20)")),
	), s.searchDownloads)

	srv.AddTool(mcpgo.NewTool("download_track",
		mcpgo.WithDescription("Download a single track by its external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("External track ID (from search_downloads results)")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
	), s.downloadTrack)

	srv.AddTool(mcpgo.NewTool("download_album",
		mcpgo.WithDescription("Download a full album by its external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcpgo.WithString("album_id", mcpgo.Required(), mcpgo.Description("External album ID (from search_downloads results)")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
	), s.downloadAlbum)

	srv.AddTool(mcpgo.NewTool("download_artist",
		mcpgo.WithDescription("Download all albums from an artist by their external ID (from search_downloads). Returns a job ID to track progress with get_job."),
		mcpgo.WithString("artist_id", mcpgo.Required(), mcpgo.Description("External artist ID (from search_downloads results)")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
	), s.downloadArtist)

	srv.AddTool(mcpgo.NewTool("download_tracks",
		mcpgo.WithDescription("Download multiple tracks at once by their external IDs. Returns a job ID. Pass IDs as a comma-separated string."),
		mcpgo.WithString("track_ids", mcpgo.Required(), mcpgo.Description("Comma-separated list of external track IDs")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
	), s.downloadTracks)

	srv.AddTool(mcpgo.NewTool("download_playlist",
		mcpgo.WithDescription("Download a set of tracks as a named playlist. Resolve an external playlist URL to track IDs first with search_downloads (type 'link'). Returns a job ID."),
		mcpgo.WithString("track_ids", mcpgo.Required(), mcpgo.Description("Comma-separated list of external track IDs")),
		mcpgo.WithString("playlist_name", mcpgo.Required(), mcpgo.Description("Name for the playlist")),
		mcpgo.WithString("downloader", mcpgo.Required(), mcpgo.Description("Downloader plugin name (e.g. 'deezer')")),
	), s.downloadPlaylist)
}

func (s *Service) getDownloadCapabilities(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	statuses := s.downloading.GetDownloaderStatuses()
	result := make(map[string]any, len(statuses))
	for name, status := range statuses {
		entry := map[string]any{
			"status":  status.Status,
			"message": status.Message,
		}
		if caps, err := s.downloading.GetDownloaderCapabilities(name); err == nil {
			entry["capabilities"] = caps
		}
		result[name] = entry
	}
	return jsonResult(result), nil
}

func (s *Service) getDownloaderInfo(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	downloader := strArg(req.Params.Arguments, "downloader")
	if downloader == "" {
		return jsonResult(s.downloading.GetDownloaderStatuses()), nil
	}
	userInfo := s.downloading.GetUserInfo(downloader)
	if userInfo == nil {
		return errResult("no user info available for downloader: " + downloader), nil
	}
	return jsonResult(userInfo), nil
}

func (s *Service) searchDownloads(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	query := strArg(args, "query")
	downloader := strArg(args, "downloader")
	limit := intArg(args, "limit", 20)

	switch strArg(args, "type") {
	case "track":
		tracks, err := s.downloading.SearchTracks(downloader, query, limit)
		if err != nil {
			return errResult(err.Error()), nil
		}
		results := make([]map[string]any, 0, len(tracks))
		for i := range tracks {
			results = append(results, trackSummary(&tracks[i]))
		}
		return jsonResult(results), nil
	case "album":
		albums, err := s.downloading.SearchAlbums(downloader, query, limit)
		if err != nil {
			return errResult(err.Error()), nil
		}
		results := make([]map[string]any, 0, len(albums))
		for i := range albums {
			results = append(results, albumSummary(&albums[i]))
		}
		return jsonResult(results), nil
	case "artist":
		artists, err := s.downloading.SearchArtists(downloader, query, limit)
		if err != nil {
			return errResult(err.Error()), nil
		}
		results := make([]map[string]any, 0, len(artists))
		for i := range artists {
			results = append(results, artistSummary(&artists[i]))
		}
		return jsonResult(results), nil
	case "link":
		result, err := s.downloading.SearchLinks(downloader, query, limit)
		if err != nil {
			return errResult(err.Error()), nil
		}
		return jsonResult(result), nil
	default:
		return errResult("type must be one of: track, album, artist, link"), nil
	}
}

func (s *Service) downloadTrack(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	jobID, err := s.downloading.DownloadTrack(strArg(args, "downloader"), strArg(args, "track_id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) downloadAlbum(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	jobID, err := s.downloading.DownloadAlbum(strArg(args, "downloader"), strArg(args, "album_id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) downloadArtist(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	jobID, err := s.downloading.DownloadArtist(strArg(args, "downloader"), strArg(args, "artist_id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

// splitIDs splits a comma-separated ID list, trimming whitespace and dropping empties.
func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}

func (s *Service) downloadTracks(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	ids := splitIDs(strArg(args, "track_ids"))
	if len(ids) == 0 {
		return errResult("track_ids is required"), nil
	}
	jobID, err := s.downloading.DownloadTracks(strArg(args, "downloader"), ids)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) downloadPlaylist(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	ids := splitIDs(strArg(args, "track_ids"))
	if len(ids) == 0 {
		return errResult("track_ids is required"), nil
	}
	playlistName := strArg(args, "playlist_name")
	if playlistName == "" {
		return errResult("playlist_name is required"), nil
	}
	jobID, err := s.downloading.DownloadPlaylist(strArg(args, "downloader"), ids, playlistName)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}
