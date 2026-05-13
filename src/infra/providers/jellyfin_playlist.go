package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/contre95/soulsolid/src/music"
)

// JellyfinPlaylistProvider implements music.PlaylistProvider for Jellyfin media servers.
type JellyfinPlaylistProvider struct {
	mediaBrowserClient
	enabled     bool
	displayName string
}

// NewJellyfinPlaylistProvider creates a new Jellyfin playlist provider.
func NewJellyfinPlaylistProvider(name, baseURL, apiKey, userID string, enabled bool) *JellyfinPlaylistProvider {
	p := &JellyfinPlaylistProvider{
		enabled:     enabled,
		displayName: name,
	}
	p.mediaBrowserClient = mediaBrowserClient{
		baseURL:    baseURL,
		userID:     userID,
		httpClient: &http.Client{},
		authHeader: func() string {
			return fmt.Sprintf("MediaBrowser Client=\"SoulSolid\", Device=\"SoulSolid\", DeviceId=\"soulsolid\", Version=\"1.0.0\", Token=\"%s\"", apiKey)
		},
	}
	return p
}

func (p *JellyfinPlaylistProvider) Name() string        { return "jellyfin" }
func (p *JellyfinPlaylistProvider) DisplayName() string { return p.displayName }
func (p *JellyfinPlaylistProvider) IsEnabled() bool     { return p.enabled }

func (p *JellyfinPlaylistProvider) ListPlaylists(ctx context.Context) ([]music.RemotePlaylist, error) {
	return p.listPlaylists(ctx)
}

func (p *JellyfinPlaylistProvider) GetPlaylist(ctx context.Context, remoteID string) (*music.RemotePlaylist, error) {
	return p.getPlaylist(ctx, remoteID)
}

func (p *JellyfinPlaylistProvider) CreatePlaylist(ctx context.Context, name, description string) (string, error) {
	return p.createPlaylist(ctx, name, description)
}

func (p *JellyfinPlaylistProvider) AddTracksToPlaylist(ctx context.Context, remotePlaylistID string, remoteTrackIDs []string) error {
	return p.addTracksToPlaylist(ctx, remotePlaylistID, remoteTrackIDs)
}

func (p *JellyfinPlaylistProvider) RemoveTracksFromPlaylist(ctx context.Context, remotePlaylistID string, entryIDs []string) error {
	return p.removeTracksFromPlaylist(ctx, remotePlaylistID, entryIDs)
}

func (p *JellyfinPlaylistProvider) FindTrackByPath(ctx context.Context, path string) (*music.RemoteTrack, error) {
	return p.findTrackByPath(ctx, path)
}

func (p *JellyfinPlaylistProvider) FindTrackByMetadata(ctx context.Context, title, artist string) (*music.RemoteTrack, error) {
	return p.findTrackByMetadata(ctx, title, artist)
}
