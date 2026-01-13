package playlists

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// PlayerProvider defines the interface for external media player integrations
type PlayerProvider interface {
	// Name returns the name of the player provider
	Name() string

	// IsEnabled returns whether this provider is enabled
	IsEnabled() bool

	// SyncPlaylist syncs a playlist to the external player
	SyncPlaylist(ctx context.Context, playlist *music.Playlist) error

	// GetPlaylists retrieves playlists from the external player
	GetPlaylists(ctx context.Context) ([]*music.Playlist, error)

	// DeletePlaylist removes a playlist from the external player
	DeletePlaylist(ctx context.Context, playlistID string) error
}
