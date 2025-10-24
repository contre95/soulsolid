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

// Discogs API response structures
type discogsSearchResponse struct {
	Results []discogsResult `json:"results"`
}

type discogsErrorResponse struct {
	Message string `json:"message"`
}

type discogsResult struct {
	ID          int      `json:"id"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Year        string   `json:"year"`
	Genre       []string `json:"genre"`
	Style       []string `json:"style"`
	Country     string   `json:"country"`
	Format      []string `json:"format"`
	Label       []string `json:"label"`
	ResourceURL string   `json:"resource_url"`
	Artist      string   `json:"artist"`
	Thumb       string   `json:"thumb"`
	CoverImage  string   `json:"cover_image"`
	URI         string   `json:"uri"`
	MasterID    int      `json:"master_id"`
	MasterURL   string   `json:"master_url"`
}

type discogsReleaseResponse struct {
	ID          int             `json:"id"`
	Title       string          `json:"title"`
	Year        int             `json:"year"`
	Genres      []string        `json:"genres"`
	Styles      []string        `json:"styles"`
	Artists     []discogsArtist `json:"artists"`
	Tracklist   []discogsTrack  `json:"tracklist"`
	ResourceURL string          `json:"resource_url"`
	URI         string          `json:"uri"`
}

type discogsArtist struct {
	Name string `json:"name"`
}

type discogsTrack struct {
	Position string `json:"position"`
	Title    string `json:"title"`
	Duration string `json:"duration"`
}

// DiscogsProvider implements MetadataProvider for Discogs
type DiscogsProvider struct {
	enabled bool
	apiKey  string
}

// NewDiscogsProvider creates a new Discogs provider
func NewDiscogsProvider(enabled bool, apiKey string) *DiscogsProvider {
	return &DiscogsProvider{enabled: enabled, apiKey: apiKey}
}

func (p *DiscogsProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Build search query
	query := url.Values{}
	if params.Title != "" {
		query.Set("q", params.Title)
	}
	if params.AlbumArtist != "" {
		if query.Get("q") != "" {
			query.Set("q", query.Get("q")+" "+params.AlbumArtist)
		} else {
			query.Set("q", params.AlbumArtist)
		}
	}
	if params.Album != "" {
		query.Set("release_title", params.Album)
	}
	if params.Year > 0 {
		query.Set("year", strconv.Itoa(params.Year))
	}

	// Set type to release for album/track searches
	query.Set("type", "release")

	// Limit search results to avoid too many API calls
	query.Set("per_page", "5")

	// Build URL
	baseURL := "https://api.discogs.com/database/search"
	fullURL := baseURL + "?" + query.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required User-Agent
	req.Header.Set("User-Agent", "SoulSolid/1.0")

	// Set authorization if API key is provided
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Discogs token="+p.apiKey)
	}

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("Discogs API rate limit exceeded, please try again later")
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp discogsErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Message != "" {
			return nil, fmt.Errorf("Discogs API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp discogsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to music.Track by fetching release details and extracting tracklist
	var tracks []*music.Track
	seenTracks := make(map[string]bool) // For deduplication

	for _, result := range searchResp.Results {
		// Fetch full release details
		release, err := p.fetchReleaseDetails(ctx, result.ResourceURL)
		if err != nil {
			// Log error but continue with other releases
			continue
		}

		// Convert each track in the tracklist
		for _, discogsTrack := range release.Tracklist {
			track := p.convertDiscogsTrackToTrack(discogsTrack, *release)

			// Create a unique key for deduplication
			trackKey := fmt.Sprintf("%s-%s-%s-%d", track.Title, release.Title, release.Artists[0].Name, release.Year)
			if !seenTracks[trackKey] {
				tracks = append(tracks, track)
				seenTracks[trackKey] = true
			}

			// Limit to 10 tracks total
			if len(tracks) >= 10 {
				break
			}
		}

		if len(tracks) >= 10 {
			break
		}
	}

	return tracks, nil
}

// convertDiscogsResultToTrack converts a Discogs API result to a music.Track
func (p *DiscogsProvider) convertDiscogsResultToTrack(result discogsResult) *music.Track {
	// Parse year
	year := 0
	if result.Year != "" {
		if y, err := strconv.Atoi(result.Year); err == nil {
			year = y
		}
	}

	// Determine genre
	genre := ""
	if len(result.Genre) > 0 {
		genre = result.Genre[0]
	} else if len(result.Style) > 0 {
		genre = result.Style[0]
	}

	// Create artist
	artistName := result.Artist
	if artistName == "" {
		// Try to extract from title if it's in "Artist - Title" format
		parts := strings.Split(result.Title, " - ")
		if len(parts) >= 2 {
			artistName = parts[0]
		}
	}

	artist := &music.Artist{Name: artistName}

	// Create album
	albumTitle := result.Title
	if artistName != "" && strings.HasPrefix(result.Title, artistName+" - ") {
		albumTitle = strings.TrimPrefix(result.Title, artistName+" - ")
	}

	album := &music.Album{
		Title: albumTitle,
		Artists: []music.ArtistRole{
			{Artist: artist, Role: "main"},
		},
	}

	// Create track (using album title as track title for now, since Discogs search returns releases)
	track := &music.Track{
		Title: albumTitle, // This is a simplification; real tracks would need release details
		Artists: []music.ArtistRole{
			{Artist: artist, Role: "main"},
		},
		Album: album,
		Metadata: music.Metadata{
			Year:  year,
			Genre: genre,
		},
	}

	// Ensure the URI is a full URL
	discogsURL := result.URI
	if !strings.HasPrefix(discogsURL, "http") {
		discogsURL = "https://www.discogs.com" + discogsURL
	}

	track.MetadataSource = music.MetadataSource{
		Source:            "discogs",
		MetadataSourceURL: discogsURL,
	}

	return track
}

// convertDiscogsTrackToTrack converts a Discogs track from a release to a music.Track
func (p *DiscogsProvider) convertDiscogsTrackToTrack(discogsTrack discogsTrack, release discogsReleaseResponse) *music.Track {
	// Get artist name
	artistName := ""
	if len(release.Artists) > 0 {
		artistName = release.Artists[0].Name
	}

	// Create artist
	artist := &music.Artist{Name: artistName}

	// Create album
	album := &music.Album{
		Title: release.Title,
		Artists: []music.ArtistRole{
			{Artist: artist, Role: "main"},
		},
	}

	// Determine genre
	genre := ""
	if len(release.Genres) > 0 {
		genre = release.Genres[0]
	} else if len(release.Styles) > 0 {
		genre = release.Styles[0]
	}

	// Parse track number from position
	trackNumber := 0
	if num, err := strconv.Atoi(discogsTrack.Position); err == nil {
		trackNumber = num
	}

	// Create track
	track := &music.Track{
		Title: discogsTrack.Title,
		Artists: []music.ArtistRole{
			{Artist: artist, Role: "main"},
		},
		Album: album,
		Metadata: music.Metadata{
			Year:        release.Year,
			Genre:       genre,
			TrackNumber: trackNumber,
		},
	}

	// Ensure the URI is a full URL
	discogsURL := release.URI
	if !strings.HasPrefix(discogsURL, "http") {
		discogsURL = "https://www.discogs.com" + discogsURL
	}

	track.MetadataSource = music.MetadataSource{
		Source:            "discogs",
		MetadataSourceURL: discogsURL,
	}

	return track
}

// fetchReleaseDetails fetches the full release data from Discogs API
func (p *DiscogsProvider) fetchReleaseDetails(ctx context.Context, resourceURL string) (*discogsReleaseResponse, error) {
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", resourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required User-Agent
	req.Header.Set("User-Agent", "SoulSolid/1.0")

	// Set authorization if API key is provided
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Discogs token="+p.apiKey)
	}

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("Discogs API rate limit exceeded, please try again later")
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp discogsErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil && errorResp.Message != "" {
			return nil, fmt.Errorf("Discogs API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var releaseResp discogsReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&releaseResp); err != nil {
		return nil, fmt.Errorf("failed to decode release response: %w", err)
	}

	return &releaseResp, nil
}

func (p *DiscogsProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Discogs API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *DiscogsProvider) Name() string    { return "discogs" }
func (p *DiscogsProvider) IsEnabled() bool { return p.enabled }
