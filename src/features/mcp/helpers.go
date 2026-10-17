package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- Registration helpers ---
//
// Every tool is registered through one of these so it carries MCP annotations
// (clients use them to decide when to ask the user for confirmation) and so
// read-only mode can drop everything that writes.

// addReadTool registers a tool that never modifies the library, files or config.
func (s *Service) addReadTool(srv *server.MCPServer, tool mcpgo.Tool, h server.ToolHandlerFunc) {
	tool.Annotations.ReadOnlyHint = mcpgo.ToBoolPtr(true)
	tool.Annotations.DestructiveHint = mcpgo.ToBoolPtr(false)
	srv.AddTool(tool, h)
}

// addWriteTool registers a tool that modifies state in an additive or reversible way.
func (s *Service) addWriteTool(srv *server.MCPServer, tool mcpgo.Tool, h server.ToolHandlerFunc) {
	if s.readOnly {
		return
	}
	tool.Annotations.ReadOnlyHint = mcpgo.ToBoolPtr(false)
	tool.Annotations.DestructiveHint = mcpgo.ToBoolPtr(false)
	srv.AddTool(tool, h)
}

// addDestructiveTool registers a tool that deletes or overwrites data irreversibly.
func (s *Service) addDestructiveTool(srv *server.MCPServer, tool mcpgo.Tool, h server.ToolHandlerFunc) {
	if s.readOnly {
		return
	}
	tool.Annotations.ReadOnlyHint = mcpgo.ToBoolPtr(false)
	tool.Annotations.DestructiveHint = mcpgo.ToBoolPtr(true)
	srv.AddTool(tool, h)
}

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

// maxTextResult caps free-form text results (file trees, logs) so a single
// call cannot flood the agent's context window.
const maxTextResult = 64 << 10 // 64 KiB

// maxListLimit caps the page size agents can request from list/search tools.
const maxListLimit = 100

// truncateText keeps the first maxTextResult bytes of s, cut at a line break,
// and notes how much was dropped.
func truncateText(s string) string {
	if len(s) <= maxTextResult {
		return s
	}
	cut := strings.LastIndexByte(s[:maxTextResult], '\n') + 1
	if cut == 0 {
		cut = maxTextResult
	}
	return s[:cut] + fmt.Sprintf("\n[truncated: %d of %d bytes shown]", cut, len(s))
}

// tailLines returns the last n lines of s.
func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return s
	}
	return fmt.Sprintf("[showing last %d of %d lines]\n", n, len(lines)) + strings.Join(lines[len(lines)-n:], "\n")
}

// limitArg reads a page-size argument, defaulting to def and clamped to [1, maxListLimit].
func limitArg(args any, key string, def int) int {
	limit := intArg(args, key, def)
	if limit < 1 {
		return def
	}
	return min(limit, maxListLimit)
}

// --- Argument helpers ---

func strArg(args any, key string) string {
	v, _ := lookupStrArg(args, key)
	return v
}

// lookupStrArg returns the argument as a string and whether it was provided.
// Models often send numbers or booleans for string parameters (e.g. "year":
// 2004), so those are formatted instead of silently dropped.
func lookupStrArg(args any, key string) (string, bool) {
	m, ok := args.(map[string]any)
	if !ok {
		return "", false
	}
	switch v := m[key].(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(v), true
	default:
		return "", false
	}
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
