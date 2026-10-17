package mcp

import (
	"context"

	"github.com/contre95/soulsolid/src/features/merge"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerMergeTools(srv *server.MCPServer) {
	s.addReadTool(srv, mcpgo.NewTool("find_merge_groups",
		mcpgo.WithDescription("Scan the library for duplicates that can be merged. Kinds 'artist', 'album' and 'genre' group entries whose names normalize to the same key; 'track' groups different files sharing an AcoustID (same recording). Each group has a key, a suggested canonical value and its variants (id, value, sub). Use merge_metadata, merge_tracks or keep_duplicate_tracks to resolve a group."),
		mcpgo.WithString("kind", mcpgo.Required(), mcpgo.Description("One of: artist, album, genre, track")),
		mcpgo.WithBoolean("include_kept", mcpgo.Description("Track groups only: also return groups previously marked as intentional duplicates (default false)")),
	), s.findMergeGroups)

	s.addDestructiveTool(srv, mcpgo.NewTool("merge_metadata",
		mcpgo.WithDescription("Merge a group of artists, albums or genres into one canonical entry, updating the database and rewriting the affected files' tags. Returns a job ID. For artists and albums pass variant IDs; for genres pass the genre strings."),
		mcpgo.WithString("kind", mcpgo.Required(), mcpgo.Description("One of: artist, album, genre")),
		mcpgo.WithString("canonical", mcpgo.Required(), mcpgo.Description("ID (artist/album) or genre string to keep; must also be listed in members")),
		mcpgo.WithString("members", mcpgo.Required(), mcpgo.Description("Comma-separated IDs (artist/album) or genre strings of the whole group, including the canonical one")),
	), s.mergeMetadata)

	s.addDestructiveTool(srv, mcpgo.NewTool("merge_tracks",
		mcpgo.WithDescription("Resolve a duplicate-track group: the tracks in remove_ids are deleted from the library AND their files deleted from disk; playlist entries are repointed to a surviving track. At least one track of the group must survive. Returns a job ID."),
		mcpgo.WithString("track_ids", mcpgo.Required(), mcpgo.Description("Comma-separated IDs of every track in the group")),
		mcpgo.WithString("remove_ids", mcpgo.Required(), mcpgo.Description("Comma-separated IDs of the tracks to delete (a subset of track_ids)")),
	), s.mergeTracks)

	s.addWriteTool(srv, mcpgo.NewTool("keep_duplicate_tracks",
		mcpgo.WithDescription("Mark a duplicate-track group as intentional (e.g. a FLAC and an MP3 kept on purpose). No files or tags change; the group stops appearing in find_merge_groups."),
		mcpgo.WithString("acoustid", mcpgo.Required(), mcpgo.Description("Group key (AcoustID) from find_merge_groups")),
		mcpgo.WithString("track_ids", mcpgo.Required(), mcpgo.Description("Comma-separated IDs of every track in the group")),
	), s.keepDuplicateTracks)
}

func (s *Service) findMergeGroups(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	var (
		groups []merge.Group
		err    error
	)
	switch merge.Kind(strArg(args, "kind")) {
	case merge.KindArtist:
		groups, err = s.merge.FindArtistGroups(ctx)
	case merge.KindAlbum:
		groups, err = s.merge.FindAlbumGroups(ctx)
	case merge.KindGenre:
		groups, err = s.merge.FindGenreGroups(ctx)
	case merge.KindTrack:
		groups, err = s.merge.FindTrackGroups(ctx, boolArg(args, "include_kept", false))
	default:
		return errResult("kind must be one of: artist, album, genre, track"), nil
	}
	if err != nil {
		return errResult(err.Error()), nil
	}

	results := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		variants := make([]map[string]string, 0, len(g.Variants))
		for _, v := range g.Variants {
			variants = append(variants, map[string]string{"id": v.ID, "value": v.Value, "sub": v.Sub})
		}
		results = append(results, map[string]any{
			"key":       g.Key,
			"canonical": g.Canonical,
			"variants":  variants,
		})
	}
	return jsonResult(results), nil
}

func (s *Service) mergeMetadata(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	kind := merge.Kind(strArg(args, "kind"))
	if kind == merge.KindTrack {
		return errResult("use merge_tracks for duplicate tracks"), nil
	}
	jobID, err := s.merge.StartMerge(ctx, kind, strArg(args, "canonical"), splitIDs(strArg(args, "members")))
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) mergeTracks(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	all := splitIDs(strArg(args, "track_ids"))
	remove := splitIDs(strArg(args, "remove_ids"))

	// StartTrackMerge deletes remove_ids as given. Require them to belong to the
	// declared group and the whole group to share one AcoustID, so this tool
	// can only remove real duplicates, never an unrelated file.
	inGroup := make(map[string]bool, len(all))
	acoustID := ""
	for _, id := range all {
		track, err := s.library.GetTrack(ctx, id)
		if err != nil {
			return errResult(err.Error()), nil
		}
		if track == nil {
			return errResult("track not found: " + id), nil
		}
		trackAcoustID := track.Attributes["acoustid"]
		if trackAcoustID == "" || (acoustID != "" && trackAcoustID != acoustID) {
			return errResult("all track_ids must share the same AcoustID (use find_merge_groups with kind 'track')"), nil
		}
		acoustID = trackAcoustID
		inGroup[id] = true
	}
	for _, id := range remove {
		if !inGroup[id] {
			return errResult("remove_ids must be a subset of track_ids: " + id), nil
		}
	}

	jobID, err := s.merge.StartTrackMerge(ctx, all, remove)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}

func (s *Service) keepDuplicateTracks(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	if err := s.merge.KeepTracks(ctx, strArg(args, "acoustid"), splitIDs(strArg(args, "track_ids"))); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("duplicate group marked as intentional"), nil
}
