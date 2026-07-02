package mcp

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerPlaylistTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("list_playlists",
		mcpgo.WithDescription("List all playlists with their track counts"),
	), s.listPlaylists)

	srv.AddTool(mcpgo.NewTool("get_playlist",
		mcpgo.WithDescription("Get a playlist and its tracks by UUID"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Playlist UUID")),
	), s.getPlaylist)

	srv.AddTool(mcpgo.NewTool("create_playlist",
		mcpgo.WithDescription("Create a new empty playlist"),
		mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("Playlist name")),
		mcpgo.WithString("description", mcpgo.Description("Optional description")),
	), s.createPlaylist)

	srv.AddTool(mcpgo.NewTool("update_playlist",
		mcpgo.WithDescription("Rename or update the description of a playlist"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Playlist UUID")),
		mcpgo.WithString("name", mcpgo.Required(), mcpgo.Description("New playlist name")),
		mcpgo.WithString("description", mcpgo.Description("New description")),
	), s.updatePlaylist)

	srv.AddTool(mcpgo.NewTool("delete_playlist",
		mcpgo.WithDescription("Permanently delete a playlist (does not delete the tracks themselves)"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Playlist UUID")),
	), s.deletePlaylist)

	srv.AddTool(mcpgo.NewTool("add_to_playlist",
		mcpgo.WithDescription("Add a track, album, or artist to a playlist. Adding an album or artist adds all their tracks."),
		mcpgo.WithString("playlist_id", mcpgo.Required(), mcpgo.Description("Playlist UUID")),
		mcpgo.WithString("item_type", mcpgo.Required(), mcpgo.Description("One of: track, album, artist")),
		mcpgo.WithString("item_id", mcpgo.Required(), mcpgo.Description("UUID of the track, album, or artist to add")),
	), s.addToPlaylist)

	srv.AddTool(mcpgo.NewTool("remove_from_playlist",
		mcpgo.WithDescription("Remove a specific track from a playlist"),
		mcpgo.WithString("playlist_id", mcpgo.Required(), mcpgo.Description("Playlist UUID")),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID to remove")),
	), s.removeFromPlaylist)

	srv.AddTool(mcpgo.NewTool("get_track_playlists",
		mcpgo.WithDescription("List all playlists that contain a given track"),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getTrackPlaylists)
}

// playlistSummary returns a compact representation of a playlist.
func playlistSummary(p *music.Playlist) map[string]any {
	return map[string]any{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"tracks":      len(p.Tracks),
	}
}

func (s *Service) listPlaylists(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	playlists, err := s.playlists.GetAllPlaylists(ctx)
	if err != nil {
		return errResult(err.Error()), nil
	}
	results := make([]map[string]any, 0, len(playlists))
	for _, p := range playlists {
		results = append(results, playlistSummary(p))
	}
	return jsonResult(results), nil
}

func (s *Service) getPlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	playlist, err := s.playlists.GetPlaylist(ctx, strArg(req.Params.Arguments, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if playlist == nil {
		return errResult("playlist not found"), nil
	}
	tracks := make([]map[string]any, 0, len(playlist.Tracks))
	for _, t := range playlist.Tracks {
		tracks = append(tracks, trackSummary(t))
	}
	result := playlistSummary(playlist)
	result["tracks"] = tracks
	return jsonResult(result), nil
}

func (s *Service) createPlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	name := strArg(args, "name")
	if name == "" {
		return errResult("name is required"), nil
	}
	playlist, err := s.playlists.CreatePlaylist(ctx, name, strArg(args, "description"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(playlistSummary(playlist)), nil
}

func (s *Service) updatePlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	name := strArg(args, "name")
	if name == "" {
		return errResult("name is required"), nil
	}
	playlist, err := s.playlists.GetPlaylist(ctx, strArg(args, "id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if playlist == nil {
		return errResult("playlist not found"), nil
	}
	playlist.Name = name
	playlist.Description = strArg(args, "description")
	if err := s.playlists.UpdatePlaylist(ctx, playlist); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("playlist updated successfully"), nil
}

func (s *Service) deletePlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.playlists.DeletePlaylist(ctx, strArg(req.Params.Arguments, "id")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("playlist deleted successfully"), nil
}

func (s *Service) addToPlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.playlists.AddItemToPlaylist(ctx, strArg(args, "playlist_id"), strArg(args, "item_type"), strArg(args, "item_id")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("added to playlist successfully"), nil
}

func (s *Service) removeFromPlaylist(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.playlists.RemoveTrackFromPlaylist(ctx, strArg(args, "playlist_id"), strArg(args, "track_id")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("removed from playlist successfully"), nil
}

func (s *Service) getTrackPlaylists(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	playlists, err := s.playlists.GetPlaylistsContainingTrack(ctx, strArg(req.Params.Arguments, "track_id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	results := make([]map[string]any, 0, len(playlists))
	for _, p := range playlists {
		results = append(results, playlistSummary(p))
	}
	return jsonResult(results), nil
}
