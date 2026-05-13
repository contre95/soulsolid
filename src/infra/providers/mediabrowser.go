package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/contre95/soulsolid/src/music"
)

// API response types shared between Emby and Jellyfin (both use the MediaBrowser protocol).
type mbItemsResponse struct {
	Items []mbItem `json:"Items"`
}

type mbItem struct {
	Id             string   `json:"Id"`
	Name           string   `json:"Name"`
	Overview       string   `json:"Overview"`
	Path           string   `json:"Path"`
	Artists        []string `json:"Artists"`
	AlbumArtist    string   `json:"AlbumArtist"`
	Album          string   `json:"Album"`
	PlaylistItemId string   `json:"PlaylistItemId"`
}

type mbCreatePlaylistResponse struct {
	Id string `json:"Id"`
}

// mediaBrowserClient is the shared HTTP base for Emby and Jellyfin providers.
// Both servers implement the MediaBrowser API; they differ only in authentication headers.
type mediaBrowserClient struct {
	baseURL    string
	userID     string
	httpClient *http.Client
	authHeader func() string
}

func (c *mediaBrowserClient) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	var req *http.Request
	var err error
	if reqBody != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.httpClient.Do(req)
}

func (c *mediaBrowserClient) listPlaylists(ctx context.Context) ([]music.RemotePlaylist, error) {
	path := fmt.Sprintf("/Items?userId=%s&IncludeItemTypes=Playlist&Recursive=true&Fields=Overview",
		url.QueryEscape(c.userID))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list playlists request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list playlists: server returned %d", resp.StatusCode)
	}

	var result mbItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode list playlists response: %w", err)
	}

	playlists := make([]music.RemotePlaylist, 0, len(result.Items))
	for _, item := range result.Items {
		playlists = append(playlists, music.RemotePlaylist{
			RemoteID:    item.Id,
			Name:        item.Name,
			Description: item.Overview,
		})
	}
	return playlists, nil
}

func (c *mediaBrowserClient) getPlaylist(ctx context.Context, remoteID string) (*music.RemotePlaylist, error) {
	path := fmt.Sprintf("/Playlists/%s/Items?userId=%s&Fields=Path,Artists,AlbumArtist,Album",
		url.PathEscape(remoteID), url.QueryEscape(c.userID))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get playlist items request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get playlist items: server returned %d", resp.StatusCode)
	}

	var result mbItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode playlist items response: %w", err)
	}

	tracks := make([]music.RemoteTrack, 0, len(result.Items))
	for _, item := range result.Items {
		artist := item.AlbumArtist
		if artist == "" && len(item.Artists) > 0 {
			artist = item.Artists[0]
		}
		tracks = append(tracks, music.RemoteTrack{
			RemoteID: item.Id,
			EntryID:  item.PlaylistItemId,
			Path:     item.Path,
			Title:    item.Name,
			Artist:   artist,
			Album:    item.Album,
		})
	}
	return &music.RemotePlaylist{RemoteID: remoteID, Tracks: tracks}, nil
}

func (c *mediaBrowserClient) createPlaylist(ctx context.Context, name, _ string) (string, error) {
	// Emby ≥ 4.8 and Jellyfin expect query params; older Emby accepted a JSON
	// body. Sending params in the query string works for all versions.
	path := fmt.Sprintf("/Playlists?Name=%s&UserId=%s&MediaType=Audio",
		url.QueryEscape(name), url.QueryEscape(c.userID))
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", fmt.Errorf("create playlist request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create playlist: server returned %d", resp.StatusCode)
	}

	var result mbCreatePlaylistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode create playlist response: %w", err)
	}
	return result.Id, nil
}

func (c *mediaBrowserClient) addTracksToPlaylist(ctx context.Context, remotePlaylistID string, remoteTrackIDs []string) error {
	if len(remoteTrackIDs) == 0 {
		return nil
	}
	path := fmt.Sprintf("/Playlists/%s/Items?ids=%s&userId=%s",
		url.PathEscape(remotePlaylistID),
		url.QueryEscape(strings.Join(remoteTrackIDs, ",")),
		url.QueryEscape(c.userID))
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("add tracks to playlist request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("add tracks to playlist: server returned %d", resp.StatusCode)
	}
	return nil
}

func (c *mediaBrowserClient) removeTracksFromPlaylist(ctx context.Context, remotePlaylistID string, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}
	path := fmt.Sprintf("/Playlists/%s/Items?entryIds=%s",
		url.PathEscape(remotePlaylistID),
		url.QueryEscape(strings.Join(entryIDs, ",")))
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("remove tracks from playlist request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remove tracks from playlist: server returned %d", resp.StatusCode)
	}
	return nil
}

func (c *mediaBrowserClient) findTrackByPath(ctx context.Context, path string) (*music.RemoteTrack, error) {
	// Extract the filename (with extension) for searching.
	filename := path
	if idx := strings.LastIndexByte(filename, '/'); idx >= 0 {
		filename = filename[idx+1:]
	}
	// Strip extension to use as searchTerm (Jellyfin/Emby don't support an exact Path filter).
	searchTerm := filename
	if idx := strings.LastIndexByte(searchTerm, '.'); idx >= 0 {
		searchTerm = searchTerm[:idx]
	}

	// Limit to 10: the filename search is specific enough that the right track
	// should rank first. Without a limit Emby/Jellyfin may return thousands of
	// items when the searchTerm is short, and the matching file may not appear.
	apiPath := fmt.Sprintf("/Items?userId=%s&IncludeItemTypes=Audio&Recursive=true&searchTerm=%s&Fields=Path,Artists,AlbumArtist,Album&Limit=10",
		url.QueryEscape(c.userID), url.QueryEscape(searchTerm))
	resp, err := c.doRequest(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("find track by path request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("find track by path: server returned %d", resp.StatusCode)
	}

	var result mbItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode find track by path response: %w", err)
	}

	// Prefer exact path match, then fall back to matching just the filename
	// (handles different mount-point prefixes between SoulSolid and the media server).
	for _, item := range result.Items {
		if item.Path == path || strings.HasSuffix(item.Path, "/"+filename) {
			artist := item.AlbumArtist
			if artist == "" && len(item.Artists) > 0 {
				artist = item.Artists[0]
			}
			return &music.RemoteTrack{
				RemoteID: item.Id,
				Path:     item.Path,
				Title:    item.Name,
				Artist:   artist,
				Album:    item.Album,
			}, nil
		}
	}
	return nil, nil
}

func (c *mediaBrowserClient) findTrackByMetadata(ctx context.Context, title, artist string) (*music.RemoteTrack, error) {
	apiPath := fmt.Sprintf("/Items?userId=%s&IncludeItemTypes=Audio&Recursive=true&searchTerm=%s&Fields=Path,Artists,AlbumArtist,Album&Limit=25",
		url.QueryEscape(c.userID), url.QueryEscape(title))
	resp, err := c.doRequest(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("find track by metadata request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("find track by metadata: server returned %d", resp.StatusCode)
	}

	var result mbItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode find track by metadata response: %w", err)
	}

	artistLower := strings.ToLower(artist)
	for _, item := range result.Items {
		itemArtist := item.AlbumArtist
		if itemArtist == "" && len(item.Artists) > 0 {
			itemArtist = item.Artists[0]
		}
		// Use Contains rather than exact equality: handles "Artist feat. X" and
		// other minor tagging differences between SoulSolid and the media server.
		if strings.Contains(strings.ToLower(itemArtist), artistLower) ||
			strings.Contains(artistLower, strings.ToLower(itemArtist)) {
			return &music.RemoteTrack{
				RemoteID: item.Id,
				Path:     item.Path,
				Title:    item.Name,
				Artist:   itemArtist,
				Album:    item.Album,
			}, nil
		}
	}
	return nil, nil
}
