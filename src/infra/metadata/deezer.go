package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/tagging"
	"github.com/contre95/soulsolid/src/music"
)

// Deezer API response structures
type deezerSearchResponse struct {
	Data []deezerTrack `json:"data"`
}

type deezerTrack struct {
	ID                    int          `json:"id"`
	Readable              bool         `json:"readable"`
	Title                 string       `json:"title"`
	TitleShort            string       `json:"title_short"`
	TitleVersion          string       `json:"title_version"`
	Link                  string       `json:"link"`
	Duration              int          `json:"duration"` // in seconds
	Artist                deezerArtist `json:"artist"`
	Album                 deezerAlbum  `json:"album"`
	ISRC                  string       `json:"isrc"`
	TrackPosition         int          `json:"track_position"`
	DiskNumber            int          `json:"disk_number"`
	Rank                  int          `json:"rank"`
	ReleaseDate           string       `json:"release_date"`
	ExplicitLyrics        bool         `json:"explicit_lyrics"`
	ExplicitContentLyrics int          `json:"explicit_content_lyrics"`
	Preview               string       `json:"preview"`
}

type deezerArtist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type deezerGenre struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Type    string `json:"type"`
}

type deezerGenres struct {
	Data []deezerGenre `json:"data"`
}

type deezerAlbum struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	ReleaseDate string       `json:"release_date"`
	Genres      deezerGenres `json:"genres"`
}

// DeezerProvider implements MetadataProvider for Deezer
type DeezerProvider struct {
	enabled bool
}

// NewDeezerProvider creates a new Deezer provider
func NewDeezerProvider(enabled bool) *DeezerProvider {
	return &DeezerProvider{enabled: enabled}
}

func (p *DeezerProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Build search query
	var queryParts []string

	if params.Title != "" {
		queryParts = append(queryParts, fmt.Sprintf("track:\"%s\"", params.Title))
	}
	if params.AlbumArtist != "" {
		queryParts = append(queryParts, fmt.Sprintf("artist:\"%s\"", params.AlbumArtist))
	}
	if params.Album != "" {
		queryParts = append(queryParts, fmt.Sprintf("album:\"%s\"", params.Album))
	}

	if len(queryParts) == 0 {
		// Return empty results if no search parameters
		return []*music.Track{}, nil
	}

	query := strings.Join(queryParts, " ")

	// Build URL
	baseURL := "https://api.deezer.com/search"
	searchURL := fmt.Sprintf("%s?q=%s&limit=10", baseURL, url.QueryEscape(query))

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Deezer API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp deezerSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to music.Track, fetching album details for genre information
	var tracks []*music.Track
	for _, deezerTrack := range searchResp.Data {
		// Fetch album details to get genre information
		albumDetails, err := p.fetchAlbumDetails(ctx, deezerTrack.Album.ID)
		if err != nil {
			// If album fetch fails, continue with basic track info (no genre)
			track := p.convertDeezerTrackToTrack(deezerTrack, nil)
			if track != nil {
				tracks = append(tracks, track)
			}
			continue
		}

		track := p.convertDeezerTrackToTrack(deezerTrack, albumDetails)
		if track != nil {
			tracks = append(tracks, track)
		}
	}

	return tracks, nil
}

// fetchAlbumDetails fetches detailed album information including genres
func (p *DeezerProvider) fetchAlbumDetails(ctx context.Context, albumID int) (*deezerAlbum, error) {
	url := fmt.Sprintf("https://api.deezer.com/album/%d", albumID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create album request: %w", err)
	}

	req.Header.Set("User-Agent", "SoulSolid/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("album API request failed with status %d", resp.StatusCode)
	}

	var album deezerAlbum
	if err := json.NewDecoder(resp.Body).Decode(&album); err != nil {
		return nil, fmt.Errorf("failed to decode album response: %w", err)
	}

	return &album, nil
}

// convertDeezerTrackToTrack converts a Deezer track to a music.Track
func (p *DeezerProvider) convertDeezerTrackToTrack(deezerTrack deezerTrack, albumDetails *deezerAlbum) *music.Track {
	// Create main artist
	mainArtist := &music.Artist{Name: deezerTrack.Artist.Name}

	// Create album
	album := &music.Album{
		Title: deezerTrack.Album.Title,
		Artists: []music.ArtistRole{
			{Artist: mainArtist, Role: "main"},
		},
	}

	// Parse year from release date (prefer album details if available)
	year := 0
	releaseDate := deezerTrack.ReleaseDate
	if albumDetails != nil && albumDetails.ReleaseDate != "" {
		releaseDate = albumDetails.ReleaseDate
	}
	if releaseDate != "" {
		if y, err := strconv.Atoi(releaseDate[:4]); err == nil {
			year = y
		}
	}

	// Extract genre from album details if available
	genre := ""
	if albumDetails != nil && len(albumDetails.Genres.Data) > 0 {
		genre = albumDetails.Genres.Data[0].Name
	}

	// Use title_short if available, otherwise use title
	title := deezerTrack.Title
	if deezerTrack.TitleShort != "" {
		title = deezerTrack.TitleShort
	}

	// Create track
	track := &music.Track{
		Title: title,
		Artists: []music.ArtistRole{
			{Artist: mainArtist, Role: "main"},
		},
		Album: album,
		Metadata: music.Metadata{
			Year:        year,
			Genre:       genre,
			TrackNumber: deezerTrack.TrackPosition,
			DiscNumber:  deezerTrack.DiskNumber,
		},
		ISRC: deezerTrack.ISRC,
		MetadataSource: music.MetadataSource{
			Source:            "deezer",
			MetadataSourceURL: deezerTrack.Link,
		},
	}

	return track
}

func (p *DeezerProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Deezer API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *DeezerProvider) Name() string    { return "deezer" }
func (p *DeezerProvider) IsEnabled() bool { return p.enabled }
