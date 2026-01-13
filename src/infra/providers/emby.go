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

// Ensure EmbyProvider implements PlayerProvider
var _ PlayerProvider = (*EmbyProvider)(nil)

// EmbyProvider implements PlayerProvider for Emby.
type EmbyProvider struct {
	URL     string
	APIKey  string
	Enabled bool
}

// NewEmbyProvider creates a new Emby provider.
func NewEmbyProvider(url, apiKey string, enabled bool) *EmbyProvider {
	return &EmbyProvider{
		URL:     strings.TrimSuffix(url, "/"),
		APIKey:  apiKey,
		Enabled: enabled,
	}
}

// IsEnabled returns whether the provider is enabled.
func (e *EmbyProvider) IsEnabled() bool {
	return e.Enabled
}

// SyncPlaylist syncs a playlist to Emby.
func (e *EmbyProvider) SyncPlaylist(ctx context.Context, playlist *music.Playlist) error {
	if !e.IsEnabled() {
		return nil
	}

	// First, check if playlist exists
	existingID, err := e.getPlaylistIDByName(ctx, playlist.Name)
	if err != nil {
		slog.Error("Failed to check existing playlist", "error", err)
		return err
	}

	if existingID != "" {
		// Update existing playlist
		return e.updatePlaylist(ctx, existingID, playlist)
	}

	// Create new playlist
	return e.createPlaylist(ctx, playlist)
}

// DeletePlaylist deletes a playlist from Emby.
func (e *EmbyProvider) DeletePlaylist(ctx context.Context, playlistID string) error {
	if !e.IsEnabled() {
		return nil
	}

	url := fmt.Sprintf("%s/Playlists/%s?api_key=%s", e.URL, playlistID, e.APIKey)
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

// GetPlaylists retrieves all playlists from Emby.
func (e *EmbyProvider) GetPlaylists(ctx context.Context) ([]*music.Playlist, error) {
	if !e.IsEnabled() {
		return []*music.Playlist{}, nil
	}

	url := fmt.Sprintf("%s/Playlists?api_key=%s", e.URL, e.APIKey)
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
		Items []struct {
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"Items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var playlists []*music.Playlist
	for _, item := range result.Items {
		playlists = append(playlists, &music.Playlist{
			ID:   item.ID,
			Name: item.Name,
		})
	}

	return playlists, nil
}

// Helper methods

func (e *EmbyProvider) getPlaylistIDByName(ctx context.Context, name string) (string, error) {
	playlists, err := e.GetPlaylists(ctx)
	if err != nil {
		return "", err
	}

	for _, p := range playlists {
		if p.Name == name {
			return p.ID, nil
		}
	}

	return "", nil
}

func (e *EmbyProvider) createPlaylist(ctx context.Context, playlist *music.Playlist) error {
	// Get item IDs for tracks
	var itemIDs []string
	for _, track := range playlist.Tracks {
		// Emby uses item IDs, we need to map track paths or IDs to Emby item IDs
		// This is a simplified version - in practice, you'd need to search Emby for matching items
		embyID, err := e.findEmbyItemID(ctx, track)
		if err != nil {
			slog.Warn("Failed to find Emby item ID for track", "track", track.Title, "error", err)
			continue
		}
		itemIDs = append(itemIDs, embyID)
	}

	if len(itemIDs) == 0 {
		return fmt.Errorf("no tracks could be mapped to Emby items")
	}

	url := fmt.Sprintf("%s/Playlists?api_key=%s", e.URL, e.APIKey)

	payload := map[string]interface{}{
		"Name":        playlist.Name,
		"ItemIds":     itemIDs,
		"Description": playlist.Description,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create playlist: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (e *EmbyProvider) updatePlaylist(ctx context.Context, playlistID string, playlist *music.Playlist) error {
	// For simplicity, delete and recreate
	err := e.DeletePlaylist(ctx, playlistID)
	if err != nil {
		return err
	}
	return e.createPlaylist(ctx, playlist)
}

func (e *EmbyProvider) findEmbyItemID(ctx context.Context, track *music.Track) (string, error) {
	// Search Emby for the track by title and artist
	artistName := ""
	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		artistName = track.Artists[0].Artist.Name
	}

	query := fmt.Sprintf("search?term=%s %s&IncludeItemTypes=Audio&api_key=%s", artistName, track.Title, e.APIKey)
	url := fmt.Sprintf("%s/%s", e.URL, query)

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
		Items []struct {
			ID string `json:"Id"`
		} `json:"Items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("no matching items found")
	}

	return result.Items[0].ID, nil
}
