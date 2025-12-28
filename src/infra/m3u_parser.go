package infra

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/features/playlists"
	"github.com/contre95/soulsolid/src/music"
)

// M3UParserImpl implements the M3UParser interface
type M3UParserImpl struct {
	playlistService music.PlaylistService
	library         music.Library
}

// NewM3UParser creates a new M3U parser
func NewM3UParser(playlistService music.PlaylistService, library music.Library) playlists.M3UParser {
	return &M3UParserImpl{
		playlistService: playlistService,
		library:         library,
	}
}

// ParseM3U parses M3U content and extracts track paths
func (p *M3UParserImpl) ParseM3U(content string) ([]string, error) {
	var paths []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Clean the path (remove quotes if present)
		path := strings.Trim(line, "\"'")
		if path != "" {
			paths = append(paths, path)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing M3U content: %w", err)
	}

	return paths, nil
}

// GenerateM3U generates M3U content from tracks
func (p *M3UParserImpl) GenerateM3U(tracks []*music.Track) (string, error) {
	var builder strings.Builder

	// Write M3U header
	builder.WriteString("#EXTM3U\n\n")

	for _, track := range tracks {
		// Write EXTINF line with duration and title
		duration := track.Metadata.Duration
		if duration <= 0 {
			duration = -1 // Unknown duration
		}

		title := track.Title
		if track.TitleVersion != "" {
			title += " (" + track.TitleVersion + ")"
		}

		// Add artist info if available
		if len(track.Artists) > 0 {
			artistNames := make([]string, len(track.Artists))
			for i, artistRole := range track.Artists {
				if artistRole.Artist != nil {
					artistNames[i] = artistRole.Artist.Name
				}
			}
			title = strings.Join(artistNames, ", ") + " - " + title
		}

		builder.WriteString(fmt.Sprintf("#EXTINF:%d,%s\n", duration, title))

		// Write file path
		builder.WriteString(track.Path)
		builder.WriteString("\n\n")
	}

	return builder.String(), nil
}

// ImportM3U imports an M3U playlist by creating a playlist and matching tracks
func (p *M3UParserImpl) ImportM3U(ctx context.Context, name, description, content string) (*music.Playlist, error) {
	// Parse the M3U content
	paths, err := p.ParseM3U(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse M3U: %w", err)
	}

	// Create the playlist
	playlist, err := p.playlistService.CreatePlaylist(ctx, name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	// Match and add tracks
	addedCount := 0
	for _, path := range paths {
		// Try to find track by exact path first
		track, err := p.library.FindTrackByPath(ctx, path)
		if err != nil {
			slog.Warn("Error finding track by path", "path", path, "error", err)
			continue
		}

		if track != nil {
			// Add track to playlist
			err = p.playlistService.AddTrackToPlaylist(ctx, playlist.ID, track.ID)
			if err != nil {
				slog.Warn("Error adding track to playlist", "trackID", track.ID, "playlistID", playlist.ID, "error", err)
				continue
			}
			addedCount++
			continue
		}

		// If exact path match fails, try filename match
		filename := filepath.Base(path)
		// This is a simplified approach - in a real implementation, you might want
		// more sophisticated matching logic
		slog.Debug("Track not found by exact path, skipping", "path", path, "filename", filename)
	}

	slog.Info("M3U import completed", "playlist", name, "totalPaths", len(paths), "addedTracks", addedCount)

	// Reload playlist with tracks
	return p.playlistService.GetPlaylist(ctx, playlist.ID)
}
