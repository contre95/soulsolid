package providers

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// PlayerProvider defines the interface for external media player providers.
type PlayerProvider interface {
	// SyncPlaylist syncs a playlist to the external player.
	SyncPlaylist(ctx context.Context, playlist *music.Playlist) error
	// DeletePlaylist deletes a playlist from the external player.
	DeletePlaylist(ctx context.Context, playlistID string) error
	// GetPlaylists retrieves all playlists from the external player.
	GetPlaylists(ctx context.Context) ([]*music.Playlist, error)
	// IsEnabled returns whether the provider is enabled.
	IsEnabled() bool
}
