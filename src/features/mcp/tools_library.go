package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerLibraryTools(srv *server.MCPServer) {
	s.addReadTool(srv, mcpgo.NewTool("search_library",
		mcpgo.WithDescription("Search the SoulSolid music library. Returns tracks, albums, and artists. Use the 'id' and 'type' from results with get_track / get_album / get_artist."),
		mcpgo.WithString("query", mcpgo.Description("Search terms — matches title, artist name, album title. Omit to browse all tracks.")),
		mcpgo.WithNumber("limit", mcpgo.Description("Max results per page (default 20)")),
		mcpgo.WithNumber("page", mcpgo.Description("Page number, 1-indexed (default 1)")),
	), s.searchLibrary)

	s.addReadTool(srv, mcpgo.NewTool("get_track",
		mcpgo.WithDescription("Get full metadata for a track by its UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getTrack)

	s.addReadTool(srv, mcpgo.NewTool("get_artist",
		mcpgo.WithDescription("Get artist details by UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Artist UUID")),
	), s.getArtist)

	s.addReadTool(srv, mcpgo.NewTool("get_album",
		mcpgo.WithDescription("Get album details by UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Album UUID")),
	), s.getAlbum)

	s.addReadTool(srv, mcpgo.NewTool("library_stats",
		mcpgo.WithDescription("Get library overview: track count, artist count, album count, total storage size in bytes"),
	), s.libraryStats)

	s.addReadTool(srv, mcpgo.NewTool("get_library_tree",
		mcpgo.WithDescription("Get the file-system tree of the music library as plain text. Large libraries are truncated; prefer search_library to find specific items."),
	), s.getLibraryTree)

	s.addReadTool(srv, mcpgo.NewTool("get_track_lyrics",
		mcpgo.WithDescription("Get the lyrics currently embedded in a track's metadata"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getTrackLyrics)

	s.addDestructiveTool(srv, mcpgo.NewTool("delete_track",
		mcpgo.WithDescription("Permanently delete a track from the library and remove its file from disk"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.deleteTrack)

	s.addDestructiveTool(srv, mcpgo.NewTool("delete_album",
		mcpgo.WithDescription("Permanently delete an album and all its tracks from the library"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Album UUID")),
	), s.deleteAlbum)

	s.addDestructiveTool(srv, mcpgo.NewTool("delete_artist",
		mcpgo.WithDescription("Permanently delete an artist and all their tracks from the library"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Artist UUID")),
	), s.deleteArtist)
}

// searchLibrary runs the same unified search as the UI: albums, then artists,
// then paginated tracks.
func (s *Service) searchLibrary(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	limit := limitArg(args, "limit", 20)
	page := max(intArg(args, "page", 1), 1)

	searchPage, err := s.library.UnifiedSearch(ctx, music.TrackFilter{TextSearch: strings.TrimSpace(strArg(args, "query"))}, limit, (page-1)*limit)
	if err != nil {
		return errResult(err.Error()), nil
	}

	results := make([]map[string]any, 0, len(searchPage.Albums)+len(searchPage.Artists)+len(searchPage.Tracks))
	for _, a := range searchPage.Albums {
		results = append(results, albumSummary(a))
	}
	for _, a := range searchPage.Artists {
		results = append(results, artistSummary(a))
	}
	for _, t := range searchPage.Tracks {
		results = append(results, trackSummary(t))
	}

	return jsonResult(map[string]any{
		"results":     results,
		"total_count": searchPage.Total,
		"page":        page,
		"limit":       limit,
	}), nil
}

func (s *Service) getTrack(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	track, err := s.library.GetTrack(ctx, strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if track == nil {
		return errResult("track not found"), nil
	}
	return jsonResult(track), nil
}

func (s *Service) getArtist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	artist, err := s.library.GetArtist(ctx, strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if artist == nil {
		return errResult("artist not found"), nil
	}
	return jsonResult(artist), nil
}

func (s *Service) getAlbum(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	album, err := s.library.GetAlbum(ctx, strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if album == nil {
		return errResult("album not found"), nil
	}
	return jsonResult(album), nil
}

func (s *Service) libraryStats(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	stats := map[string]any{}
	if count, err := s.library.GetTracksCount(ctx); err == nil {
		stats["tracks"] = count
	} else {
		stats["tracks"] = "error: " + err.Error()
	}
	if count, err := s.library.GetArtistsCount(ctx); err == nil {
		stats["artists"] = count
	} else {
		stats["artists"] = "error: " + err.Error()
	}
	if count, err := s.library.GetAlbumsCount(ctx); err == nil {
		stats["albums"] = count
	} else {
		stats["albums"] = "error: " + err.Error()
	}
	if size, err := s.library.GetStorageSize(ctx); err == nil {
		stats["storage_bytes"] = size
	} else {
		stats["storage_bytes"] = "error: " + err.Error()
	}
	return jsonResult(stats), nil
}

func (s *Service) getLibraryTree(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	tree, err := s.library.GetLibraryFileTree()
	if err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(truncateText(tree)), nil
}

func (s *Service) getTrackLyrics(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	track, err := s.library.GetTrack(ctx, strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if track == nil {
		return errResult("track not found"), nil
	}
	if track.Metadata.Lyrics == "" {
		return textResult("No lyrics embedded in this track."), nil
	}
	return textResult(track.Metadata.Lyrics), nil
}

func (s *Service) deleteTrack(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	id := strArg(req.Params.Arguments, "id")
	if err := s.library.DeleteTrack(ctx, id); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(fmt.Sprintf("track %s deleted successfully", id)), nil
}

func (s *Service) deleteAlbum(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	id := strArg(req.Params.Arguments, "id")
	if err := s.library.DeleteAlbum(ctx, id); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(fmt.Sprintf("album %s deleted successfully", id)), nil
}

func (s *Service) deleteArtist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	id := strArg(req.Params.Arguments, "id")
	if err := s.library.DeleteArtist(ctx, id); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult(fmt.Sprintf("artist %s deleted successfully", id)), nil
}
