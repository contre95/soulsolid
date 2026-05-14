package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/contre95/soulsolid/src/music"
)

// EmbyPlaylistProvider implements music.PlaylistProvider for Emby media servers.
type EmbyPlaylistProvider struct {
	mediaBrowserClient
	enabled     bool
	displayName string
}

// NewEmbyPlaylistProvider creates a new Emby playlist provider.
func NewEmbyPlaylistProvider(name, baseURL, apiKey, userID string, enabled bool) *EmbyPlaylistProvider {
	p := &EmbyPlaylistProvider{
		enabled:     enabled,
		displayName: name,
	}
	p.mediaBrowserClient = mediaBrowserClient{
		baseURL:    baseURL,
		userID:     userID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authHeader: func() string {
			return fmt.Sprintf("MediaBrowser Client=\"SoulSolid\", Device=\"SoulSolid\", DeviceId=\"soulsolid\", Version=\"1.0.0\", Token=\"%s\"", apiKey)
		},
	}
	return p
}

func (p *EmbyPlaylistProvider) Name() string        { return "emby" }
func (p *EmbyPlaylistProvider) DisplayName() string { return p.displayName }
func (p *EmbyPlaylistProvider) IsEnabled() bool     { return p.enabled }

func (p *EmbyPlaylistProvider) ListPlaylists(ctx context.Context) ([]music.RemotePlaylist, error) {
	return p.listPlaylists(ctx)
}

func (p *EmbyPlaylistProvider) GetPlaylist(ctx context.Context, remoteID string) (*music.RemotePlaylist, error) {
	return p.getPlaylist(ctx, remoteID)
}

func (p *EmbyPlaylistProvider) CreatePlaylist(ctx context.Context, name, description string) (string, error) {
	return p.createPlaylist(ctx, name, description)
}

func (p *EmbyPlaylistProvider) AddTracksToPlaylist(ctx context.Context, remotePlaylistID string, remoteTrackIDs []string) error {
	return p.addTracksToPlaylist(ctx, remotePlaylistID, remoteTrackIDs)
}

func (p *EmbyPlaylistProvider) RemoveTracksFromPlaylist(ctx context.Context, remotePlaylistID string, entryIDs []string) error {
	return p.removeTracksFromPlaylist(ctx, remotePlaylistID, entryIDs)
}

func (p *EmbyPlaylistProvider) FindTrackByPath(ctx context.Context, path string) (*music.RemoteTrack, error) {
	return p.findTrackByPath(ctx, path)
}

func (p *EmbyPlaylistProvider) FindTrackByMetadata(ctx context.Context, title, artist string) (*music.RemoteTrack, error) {
	return p.findTrackByMetadata(ctx, title, artist)
}
