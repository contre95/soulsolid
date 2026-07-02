package mcp

import (
	"context"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerMetadataTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("update_track_tags",
		mcpgo.WithDescription("Save tag metadata to a track's file and the library database. Only provided fields are updated; omit a field to leave it unchanged. Artists and albums are referenced by UUID (use search_library to find them)."),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
		mcpgo.WithString("title", mcpgo.Description("Track title")),
		mcpgo.WithString("artist_ids", mcpgo.Description("Comma-separated artist UUIDs to set as the track's artists")),
		mcpgo.WithString("album_id", mcpgo.Description("Album UUID to assign the track to")),
		mcpgo.WithString("album_artist_id", mcpgo.Description("Artist UUID to set as the album artist")),
		mcpgo.WithString("year", mcpgo.Description("Release year")),
		mcpgo.WithString("genre", mcpgo.Description("Genre")),
		mcpgo.WithString("track_number", mcpgo.Description("Track number (e.g. '1')")),
		mcpgo.WithString("disc_number", mcpgo.Description("Disc number")),
		mcpgo.WithString("composer", mcpgo.Description("Composer")),
		mcpgo.WithString("isrc", mcpgo.Description("ISRC code (max 12 characters)")),
		mcpgo.WithString("lyrics", mcpgo.Description("Embedded lyrics text")),
	), s.updateTrackTags)

	srv.AddTool(mcpgo.NewTool("search_metadata_provider",
		mcpgo.WithDescription("Search an external metadata provider (e.g. musicbrainz, discogs, deezer) for tag data matching a track. Returns candidate results with their index; use get_provider_track to preview one."),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
		mcpgo.WithString("provider", mcpgo.Required(), mcpgo.Description("Metadata provider name (e.g. 'musicbrainz')")),
	), s.searchMetadataProvider)

	srv.AddTool(mcpgo.NewTool("get_provider_track",
		mcpgo.WithDescription("Preview the tag data a provider result would produce for a track, merged with the track's current data. Use after search_metadata_provider; nothing is saved — apply changes with update_track_tags."),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
		mcpgo.WithString("provider", mcpgo.Required(), mcpgo.Description("Metadata provider name")),
		mcpgo.WithNumber("index", mcpgo.Description("Index of the search result to preview (default 0)")),
	), s.getProviderTrack)

	srv.AddTool(mcpgo.NewTool("fingerprint_track",
		mcpgo.WithDescription("Generate and store an acoustic fingerprint (Chromaprint) and AcoustID for a track"),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.fingerprintTrack)

	srv.AddTool(mcpgo.NewTool("get_fingerprint",
		mcpgo.WithDescription("Get the stored acoustic fingerprint string for a track"),
		mcpgo.WithString("track_id", mcpgo.Required(), mcpgo.Description("Track UUID")),
	), s.getFingerprint)

	srv.AddTool(mcpgo.NewTool("run_acoustid_analysis",
		mcpgo.WithDescription("Start a background job that fingerprints all unidentified tracks and looks them up via AcoustID. Returns a job ID."),
	), s.runAcoustidAnalysis)
}

// formDataFromTrack pre-fills the form-field map metadata.Service.UpdateTrackTags
// expects with the track's current values, so an MCP update only changes the
// fields the caller provides (the UI always submits the full form).
func formDataFromTrack(t *music.Track) map[string]string {
	f := map[string]string{
		"title":         t.Title,
		"year":          strconv.Itoa(t.Metadata.Year),
		"genre":         t.Metadata.Genre,
		"track_number":  strconv.Itoa(t.Metadata.TrackNumber),
		"disc_number":   strconv.Itoa(t.Metadata.DiscNumber),
		"composer":      t.Metadata.Composer,
		"lyrics":        t.Metadata.Lyrics,
		"isrc":          t.ISRC,
		"title_version": t.TitleVersion,
		"source":        t.MetadataSource.Source,
		"source_url":    t.MetadataSource.MetadataSourceURL,
	}
	if t.Metadata.BPM != 0 {
		f["bpm"] = strconv.FormatFloat(t.Metadata.BPM, 'f', -1, 64)
	}
	if t.Metadata.Gain != 0 {
		f["gain"] = strconv.FormatFloat(t.Metadata.Gain, 'f', -1, 64)
	}
	if t.HasLyrics {
		f["has_lyrics"] = "true"
	} else {
		f["has_lyrics"] = "false"
	}
	var artistIDs []string
	for _, r := range t.Artists {
		if r.Artist != nil && r.Artist.ID != "" {
			artistIDs = append(artistIDs, r.Artist.ID)
		}
	}
	f["artist_ids"] = strings.Join(artistIDs, ",")
	if t.Album != nil {
		f["album_id"] = t.Album.ID
	}
	return f
}

func (s *Service) updateTrackTags(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	trackID := strArg(args, "track_id")

	track, err := s.library.GetTrack(ctx, trackID)
	if err != nil {
		return errResult(err.Error()), nil
	}
	if track == nil {
		return errResult("track not found"), nil
	}

	formData := formDataFromTrack(track)
	for _, key := range []string{"title", "artist_ids", "album_id", "album_artist_id", "year", "genre", "track_number", "disc_number", "composer", "isrc", "lyrics"} {
		if v := strArg(args, key); v != "" {
			formData[key] = v
		}
	}
	if strArg(args, "lyrics") != "" {
		formData["has_lyrics"] = "true"
	}

	if err := s.metadata.UpdateTrackTags(ctx, trackID, formData); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("tags updated successfully"), nil
}

func (s *Service) searchMetadataProvider(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	tracks, err := s.metadata.SearchTrackMetadata(ctx, strArg(args, "track_id"), strArg(args, "provider"))
	if err != nil {
		return errResult(err.Error()), nil
	}

	results := make([]map[string]any, 0, len(tracks))
	for i, t := range tracks {
		result := trackSummary(t)
		result["index"] = i
		result["year"] = t.Metadata.Year
		result["genre"] = t.Metadata.Genre
		result["source_url"] = t.MetadataSource.MetadataSourceURL
		delete(result, "id") // provider results are not library tracks
		results = append(results, result)
	}
	return jsonResult(results), nil
}

func (s *Service) getProviderTrack(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.Params.Arguments
	trackID := strArg(args, "track_id")
	index := intArg(args, "index", 0)

	tracks, err := s.metadata.SearchTrackMetadata(ctx, trackID, strArg(args, "provider"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if index < 0 || index >= len(tracks) {
		return errResult("invalid result index"), nil
	}

	currentTrack, err := s.metadata.GetTrackFileTags(ctx, trackID)
	if err != nil {
		return errResult(err.Error()), nil
	}

	merged := s.metadata.MergeFetchedData(currentTrack, tracks[index])
	return jsonResult(merged), nil
}

func (s *Service) fingerprintTrack(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	if err := s.metadata.AddChromaprintAndAcoustID(ctx, strArg(req.Params.Arguments, "track_id")); err != nil {
		return errResult(err.Error()), nil
	}
	return textResult("fingerprint calculated and saved"), nil
}

func (s *Service) getFingerprint(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	track, err := s.library.GetTrack(ctx, strArg(req.Params.Arguments, "track_id"))
	if err != nil {
		return errResult(err.Error()), nil
	}
	if track == nil {
		return errResult("track not found"), nil
	}
	if track.ChromaprintFingerprint == "" {
		return textResult("No fingerprint stored for this track. Use fingerprint_track to generate one."), nil
	}
	return textResult(track.ChromaprintFingerprint), nil
}

func (s *Service) runAcoustidAnalysis(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	jobID, err := s.metadata.StartAcoustIDAnalysis(ctx)
	if err != nil {
		return errResult(err.Error()), nil
	}
	return jsonResult(map[string]string{"job_id": jobID}), nil
}
