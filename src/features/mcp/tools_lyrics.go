package mcp

import (
	"context"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerLyricsTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("fetch_lyrics",
		mcpgo.WithDescription("Fetch lyrics for a track from an external provider (e.g. lrclib). Returns the lyrics text without saving — use update_track_tags to embed them."),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
		mcpgo.WithString("provider", mcpgo.Required(), mcpgo.Description("Lyrics provider name (e.g. 'lrclib')")),
	), s.fetchLyrics)

	srv.AddTool(mcpgo.NewTool("run_lyrics_analysis",
		mcpgo.WithDescription("Start a background job that fetches and embeds lyrics for tracks in the library. Returns a job ID."),
		mcpgo.WithString("provider", mcpgo.Required(), mcpgo.Description("Lyrics provider name (e.g. 'lrclib')")),
		mcpgo.WithBoolean("skip_existing", mcpgo.Description("Skip tracks that already have lyrics (default true)")),
		mcpgo.WithBoolean("override_no_queue", mcpgo.Description("Write lyrics directly instead of queueing conflicts for review (default false)")),
	), s.runLyricsAnalysis)

	srv.AddTool(mcpgo.NewTool("list_lyrics_queue",
		mcpgo.WithDescription("List all tracks in the lyrics review queue (tracks where fetched lyrics need approval before saving)"),
	), s.listLyricsQueue)

	srv.AddTool(mcpgo.NewTool("get_new_lyrics",
		mcpgo.WithDescription("Preview the fetched lyrics for a lyrics queue item before accepting or rejecting it"),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Lyrics queue item ID (from list_lyrics_queue)")),
	), s.getNewLyrics)

	srv.AddTool(mcpgo.NewTool("resolve_lyrics_item",
		mcpgo.WithDescription("Resolve a lyrics queue item. Actions: 'accept' (embed the new lyrics), 'reject' (discard), 'skip'."),
		mcpgo.WithString("id", mcpgo.Required(), mcpgo.Description("Lyrics queue item ID")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: accept, reject, skip")),
	), s.resolveLyricsItem)

	srv.AddTool(mcpgo.NewTool("resolve_lyrics_group",
		mcpgo.WithDescription("Apply an action to a whole group of lyrics queue items at once (e.g. all tracks from the same album)."),
		mcpgo.WithString("group_type", mcpgo.Required(), mcpgo.Description("Group type: 'album' or 'artist'")),
		mcpgo.WithString("group_key", mcpgo.Required(), mcpgo.Description("Group key value (album title or artist name)")),
		mcpgo.WithString("action", mcpgo.Required(), mcpgo.Description("One of: accept, reject, skip")),
	), s.resolveLyricsGroup)

	srv.AddTool(mcpgo.NewTool("clear_lyrics_queue",
		mcpgo.WithDescription("Discard all items from the lyrics review queue"),
	), s.clearLyricsQueue)
}

func (s *Service) fetchLyrics(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	lyricsText, err := s.lyrics.SearchLyrics(ctx, strArg(args, "track_id"), strArg(args, "provider"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if lyricsText == "" {
		return textResult("No lyrics found for this track."), nil
	}
	return textResult(lyricsText), nil
}

func (s *Service) runLyricsAnalysis(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	provider := strArg(args, "provider")
	skipExisting := boolArg(args, "skip_existing", true)
	overrideNoQueue := boolArg(args, "override_no_queue", false)

	jobID, err := s.lyrics.StartLyricsAnalysis(ctx, provider, skipExisting, overrideNoQueue)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) listLyricsQueue(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	items := s.lyrics.GetLyricsQueueItems()
	results := make([]map[string]any, 0, len(items))
	for _, item := range items {
		results = append(results, queueItemSummary(item))
	}
	return jsonResult(results), nil
}

func (s *Service) getNewLyrics(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	itemID := strArg(req.Params.Arguments, "id")
	items := s.lyrics.GetLyricsQueueItems()
	item, ok := items[itemID]
	if !ok {
		return errResult("queue item not found"), nil
	}
	if newLyrics, ok := item.Metadata["new_lyrics"]; ok && newLyrics != "" {
		return textResult(newLyrics), nil
	}
	return textResult("No new lyrics available for this item."), nil
}

func (s *Service) resolveLyricsItem(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.lyrics.ProcessLyricsQueueItem(ctx, strArg(args, "id"), strArg(args, "action")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("lyrics queue item resolved"), nil
}

func (s *Service) resolveLyricsGroup(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.lyrics.ProcessLyricsQueueGroup(ctx, strArg(args, "group_key"), strArg(args, "group_type"), strArg(args, "action")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("lyrics queue group resolved"), nil
}

func (s *Service) clearLyricsQueue(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.lyrics.ClearLyricsQueue(); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("lyrics queue cleared"), nil
}
