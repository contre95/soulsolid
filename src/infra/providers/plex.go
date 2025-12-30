package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/contre95/soulsolid/src/music"
)

// Ensure PlexProvider implements PlayerProvider
var _ PlayerProvider = (*PlexProvider)(nil)

// PlexProvider implements PlayerProvider for Plex.
type PlexProvider struct {
	URL     string
	Token   string
	Enabled bool
}

// NewPlexProvider creates a new Plex provider.
func NewPlexProvider(url, token string, enabled bool) *PlexProvider {
	return &PlexProvider{
		URL:     strings.TrimSuffix(url, "/"),
		Token:   token,
		Enabled: enabled,
	}
}

// IsEnabled returns whether the provider is enabled.
func (p *PlexProvider) IsEnabled() bool {
	return p.Enabled
}

// SyncPlaylist syncs a playlist to Plex.
func (p *PlexProvider) SyncPlaylist(ctx context.Context, playlist *music.Playlist) error {
	if !p.IsEnabled() {
		return nil
	}

	// First, check if playlist exists
	existingID, err := p.getPlaylistIDByName(ctx, playlist.Name)
	if err != nil {
		slog.Error("Failed to check existing playlist", "error", err)
		return err
	}

	if existingID != "" {
		// Update existing playlist
		return p.updatePlaylist(ctx, existingID, playlist)
	}

	// Create new playlist
	return p.createPlaylist(ctx, playlist)
}

// DeletePlaylist deletes a playlist from Plex.
func (p *PlexProvider) DeletePlaylist(ctx context.Context, playlistID string) error {
	if !p.IsEnabled() {
		return nil
	}

	url := fmt.Sprintf("%s/playlists/%s?X-Plex-Token=%s", p.URL, playlistID, p.Token)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete playlist: %s", resp.Status)
	}

	return nil
}

// GetPlaylists retrieves all playlists from Plex.
func (p *PlexProvider) GetPlaylists(ctx context.Context) ([]*music.Playlist, error) {
	if !p.IsEnabled() {
		return []*music.Playlist{}, nil
	}

	url := fmt.Sprintf("%s/playlists?X-Plex-Token=%s", p.URL, p.Token)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get playlists: %s", resp.Status)
	}

	var result struct {
		MediaContainer struct {
			Metadata []struct {
				Key   string `json:"key"`
				Title string `json:"title"`
			} `json:"Metadata"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var playlists []*music.Playlist
	for _, item := range result.MediaContainer.Metadata {
		// Extract ID from key (format: "/playlists/123")
		parts := strings.Split(item.Key, "/")
		if len(parts) >= 3 {
			id := parts[len(parts)-1]
			playlists = append(playlists, &music.Playlist{
				ID:   id,
				Name: item.Title,
			})
		}
	}

	return playlists, nil
}

// Helper methods

func (p *PlexProvider) getPlaylistIDByName(ctx context.Context, name string) (string, error) {
	playlists, err := p.GetPlaylists(ctx)
	if err != nil {
		return "", err
	}

	for _, pl := range playlists {
		if pl.Name == name {
			return pl.ID, nil
		}
	}

	return "", nil
}

func (p *PlexProvider) createPlaylist(ctx context.Context, playlist *music.Playlist) error {
	// Get rating keys for tracks
	var ratingKeys []string
	for _, track := range playlist.Tracks {
		// Plex uses rating keys, we need to map track paths or IDs to Plex rating keys
		// This is a simplified version - in practice, you'd need to search Plex for matching items
		ratingKey, err := p.findPlexRatingKey(ctx, track)
		if err != nil {
			slog.Warn("Failed to find Plex rating key for track", "track", track.Title, "error", err)
			continue
		}
		ratingKeys = append(ratingKeys, ratingKey)
	}

	if len(ratingKeys) == 0 {
		return fmt.Errorf("no tracks could be mapped to Plex items")
	}

	url := fmt.Sprintf("%s/playlists?type=audio&title=%s&smart=0&uri=%s&X-Plex-Token=%s",
		p.URL, playlist.Name, p.buildURI(ratingKeys), p.Token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(""))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create playlist: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (p *PlexProvider) updatePlaylist(ctx context.Context, playlistID string, playlist *music.Playlist) error {
	// For simplicity, delete and recreate
	err := p.DeletePlaylist(ctx, playlistID)
	if err != nil {
		return err
	}
	return p.createPlaylist(ctx, playlist)
}

func (p *PlexProvider) findPlexRatingKey(ctx context.Context, track *music.Track) (string, error) {
	// Search Plex for the track by title and artist
	artistName := ""
	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		artistName = track.Artists[0].Artist.Name
	}

	query := fmt.Sprintf("search?query=%s %s&type=10&X-Plex-Token=%s", artistName, track.Title, p.Token)
	url := fmt.Sprintf("%s/%s", p.URL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search failed: %s", resp.Status)
	}

	var result struct {
		MediaContainer struct {
			Metadata []struct {
				RatingKey string `json:"ratingKey"`
			} `json:"Metadata"`
		} `json:"MediaContainer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.MediaContainer.Metadata) == 0 {
		return "", fmt.Errorf("no matching items found")
	}

	return result.MediaContainer.Metadata[0].RatingKey, nil
}

func (p *PlexProvider) buildURI(ratingKeys []string) string {
	// Build Plex URI for playlist creation
	uris := make([]string, len(ratingKeys))
	for i, key := range ratingKeys {
		uris[i] = fmt.Sprintf("server://server/library/metadata/%s", key)
	}
	return strings.Join(uris, ",")
}
