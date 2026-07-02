package mcp

import (
	"context"
	"fmt"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerLibraryTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("search_library",
		mcpgo.WithDescription("Search the SoulSolid music library. Returns tracks, albums, and artists. Use the 'id' and 'type' from results with get_track / get_album / get_artist."),
		mcpgo.WithString("query", mcpgo.Description("Search terms — matches title, artist name, album title. Omit to browse all tracks.")),
		mcpgo.WithNumber("limit", mcpgo.Description("Max results per page (default 20)")),
		mcpgo.WithNumber("page", mcpgo.Description("Page number, 1-indexed (default 1)")),
	), s.searchLibrary)

	srv.AddTool(mcpgo.NewTool("get_track",
		mcpgo.WithDescription("Get full metadata for a track by its UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getTrack)

	srv.AddTool(mcpgo.NewTool("get_artist",
		mcpgo.WithDescription("Get artist details by UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Artist UUID")),
	), s.getArtist)

	srv.AddTool(mcpgo.NewTool("get_album",
		mcpgo.WithDescription("Get album details by UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Album UUID")),
	), s.getAlbum)

	srv.AddTool(mcpgo.NewTool("library_stats",
		mcpgo.WithDescription("Get library overview: track count, artist count, album count, total storage size in bytes"),
	), s.libraryStats)

	srv.AddTool(mcpgo.NewTool("get_library_tree",
		mcpgo.WithDescription("Get the full file-system tree of the music library as a plain text string"),
	), s.getLibraryTree)

	srv.AddTool(mcpgo.NewTool("get_track_lyrics",
		mcpgo.WithDescription("Get the lyrics currently embedded in a track's metadata"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getTrackLyrics)

	srv.AddTool(mcpgo.NewTool("delete_track",
		mcpgo.WithDescription("Permanently delete a track from the library and remove its file from disk"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.deleteTrack)

	srv.AddTool(mcpgo.NewTool("delete_album",
		mcpgo.WithDescription("Permanently delete an album and all its tracks from the library"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Album UUID")),
	), s.deleteAlbum)

	srv.AddTool(mcpgo.NewTool("delete_artist",
		mcpgo.WithDescription("Permanently delete an artist and all their tracks from the library"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Artist UUID")),
	), s.deleteArtist)
}

// searchLibrary mirrors the UI's unified search (library.Handler.GetUnifiedSearch):
// albums → artists → tracks, with artist/album matches capped and tracks paginated.
func (s *Service) searchLibrary(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	query := strArg(args, "query")
	limit := intArg(args, "limit", 20)
	page := max(intArg(args, "page", 1), 1)
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var results []map[string]any
	var totalCount int

	if query == "" {
		// Browse-all: paginated tracks only.
		tracksCount, err := s.library.GetTracksCount(ctx)
		if err != nil {
			return errResult("failed to count tracks: " + err.Error()), nil
		}
		totalCount = tracksCount
		if offset < tracksCount {
			trackLimit := min(offset+limit, tracksCount) - offset
			if trackLimit > 0 {
				tracks, err := s.library.GetTracksPaginated(ctx, trackLimit, offset)
				if err != nil {
					return errResult("failed to load tracks: " + err.Error()), nil
				}
				for _, t := range tracks {
					results = append(results, trackSummary(t))
				}
			}
		}
	} else {
		trackFilter := &music.TrackFilter{TextSearch: query}
		trackCount, err := s.library.GetTracksFilteredCount(ctx, trackFilter)
		if err != nil {
			return errResult("failed to count tracks: " + err.Error()), nil
		}

		albums, err := s.library.SearchAlbums(ctx, query, 20, 0)
		if err != nil {
			albums = nil
		}
		artists, err := s.library.GetArtistsFilteredPaginated(ctx, 20, 0, query)
		if err != nil {
			artists = nil
		}

		albumsCount := len(albums)
		artistsCount := len(artists)
		totalCount = albumsCount + artistsCount + trackCount

		start := offset
		end := min(offset+limit, totalCount)

		// Albums: [0, albumsCount)
		if start < albumsCount {
			albumEnd := min(end, albumsCount)
			for i := start; i < albumEnd; i++ {
				results = append(results, albumSummary(albums[i]))
			}
		}

		// Artists: [albumsCount, albumsCount+artistsCount)
		if end > albumsCount {
			artistStart := max(0, start-albumsCount)
			artistEnd := min(end-albumsCount, artistsCount)
			for i := artistStart; i < artistEnd; i++ {
				results = append(results, artistSummary(artists[i]))
			}
		}

		// Tracks: [albumsCount+artistsCount, totalCount)
		trackOffset := albumsCount + artistsCount
		if end > trackOffset {
			trackStart := max(0, start-trackOffset)
			trackLimit := (end - trackOffset) - trackStart
			if trackLimit > 0 {
				tracks, err := s.library.GetTracksFilteredPaginated(ctx, trackLimit, trackStart, trackFilter)
				if err == nil {
					for _, t := range tracks {
						results = append(results, trackSummary(t))
					}
				}
			}
		}
	}

	return jsonResult(map[string]any{
		"results":     results,
		"total_count": totalCount,
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
	return textResult(tree), nil
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
