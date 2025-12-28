package playlists

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// M3UParser defines the interface for M3U playlist parsing and generation
type M3UParser interface {
	// ParseM3U parses an M3U file content and returns track paths
	ParseM3U(content string) ([]string, error)

	// GenerateM3U generates M3U content from tracks
	GenerateM3U(tracks []*music.Track) (string, error)

	// ImportM3U imports an M3U playlist by matching tracks and creating a playlist
	ImportM3U(ctx context.Context, name, description, content string) (*music.Playlist, error)
}
