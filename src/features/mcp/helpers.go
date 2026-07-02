package mcp

import (
	"encoding/json"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// --- Result helpers ---

func jsonResult(v any) *mcpgo.CallToolResult {
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcpgo.NewToolResultError("failed to serialize result: " + err.Error())
	}
	return mcpgo.NewToolResultText(string(pretty))
}

func textResult(s string) *mcpgo.CallToolResult {
	return mcpgo.NewToolResultText(s)
}

func errResult(msg string) *mcpgo.CallToolResult {
	return mcpgo.NewToolResultError(msg)
}

// --- Argument helpers ---

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

func boolArg(args any, key string, def bool) bool {
	m, ok := args.(map[string]any)
	if !ok {
		return def
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return def
}

// --- Domain serialization helpers ---

// artistNames joins the names of all artists in a role slice.
func artistNames(roles []music.ArtistRole) string {
	var b strings.Builder
	for _, r := range roles {
		if r.Artist == nil {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(", ")
		}
		b.WriteString(r.Artist.Name)
	}
	return b.String()
}

// trackSummary returns a compact representation of a track for list results.
func trackSummary(t *music.Track) map[string]any {
	albumTitle := ""
	if t.Album != nil {
		albumTitle = t.Album.Title
	}
	return map[string]any{
		"id":       t.ID,
		"type":     "track",
		"title":    t.Title,
		"artists":  artistNames(t.Artists),
		"album":    albumTitle,
		"duration": t.Metadata.Duration,
	}
}

// albumSummary returns a compact representation of an album for list results.
func albumSummary(a *music.Album) map[string]any {
	year := 0
	if !a.ReleaseDate.IsZero() {
		year = a.ReleaseDate.Year()
	}
	return map[string]any{
		"id":      a.ID,
		"type":    "album",
		"title":   a.Title,
		"artists": artistNames(a.Artists),
		"year":    year,
	}
}

// artistSummary returns a compact representation of an artist for list results.
func artistSummary(a *music.Artist) map[string]any {
	return map[string]any{
		"id":   a.ID,
		"type": "artist",
		"name": a.Name,
	}
}

// queueItemSummary returns a compact representation of a queue item.
func queueItemSummary(item music.QueueItem) map[string]any {
	out := map[string]any{
		"id":        item.ID,
		"types":     item.Types,
		"job_id":    item.JobID,
		"timestamp": item.Timestamp,
	}
	if item.Track != nil {
		albumTitle := ""
		if item.Track.Album != nil {
			albumTitle = item.Track.Album.Title
		}
		out["track_id"] = item.Track.ID
		out["title"] = item.Track.Title
		out["artists"] = artistNames(item.Track.Artists)
		out["album"] = albumTitle
	}
	if len(item.Metadata) > 0 {
		meta := make(map[string]string, len(item.Metadata))
		for k, v := range item.Metadata {
			// New lyrics can be huge; expose their presence, fetch via get_new_lyrics.
			if k == "new_lyrics" {
				meta["has_new_lyrics"] = "true"
				continue
			}
			meta[k] = v
		}
		out["metadata"] = meta
	}
	return out
}

// jobSummary returns a compact representation of a job for list results.
func jobSummary(j *music.Job) map[string]any {
	return map[string]any{
		"id":         j.ID,
		"type":       j.Type,
		"name":       j.Name,
		"status":     j.Status,
		"progress":   j.Progress,
		"message":    j.Message,
		"error":      j.Error,
		"created_at": j.CreatedAt,
		"updated_at": j.UpdatedAt,
	}
}
