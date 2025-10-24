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

// MusicBrainz API response structures
type mbRecordingSearchResponse struct {
	Recordings []mbRecording `json:"recordings"`
}

type mbRecording struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Length       int              `json:"length"` // in milliseconds
	ArtistCredit []mbArtistCredit `json:"artist-credit"`
	Releases     []mbRelease      `json:"releases"`
	ISRCs        []string         `json:"isrcs"`
}

type mbArtistCredit struct {
	Name   string   `json:"name"`
	Artist mbArtist `json:"artist"`
}

type mbArtist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Disambiguation string `json:"disambiguation"`
}

type mbRelease struct {
	ID    string     `json:"id"`
	Title string     `json:"title"`
	Date  string     `json:"date"`
	Media []mbMedium `json:"media"`
}

type mbMedium struct {
	Tracks []mbTrack `json:"tracks"`
}

type mbTrack struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Length int    `json:"length"`
}

// MusicBrainzProvider implements MetadataProvider for MusicBrainz
type MusicBrainzProvider struct {
	enabled bool
}

// NewMusicBrainzProvider creates a new MusicBrainz provider
func NewMusicBrainzProvider(enabled bool) *MusicBrainzProvider {
	return &MusicBrainzProvider{enabled: enabled}
}

func (p *MusicBrainzProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Build search query using Lucene syntax
	var queryParts []string

	if params.Title != "" {
		queryParts = append(queryParts, fmt.Sprintf("recording:\"%s\"", params.Title))
	}
	if params.AlbumArtist != "" {
		queryParts = append(queryParts, fmt.Sprintf("artist:\"%s\"", params.AlbumArtist))
	}
	if params.Album != "" {
		queryParts = append(queryParts, fmt.Sprintf("release:\"%s\"", params.Album))
	}
	if params.Year > 0 {
		queryParts = append(queryParts, fmt.Sprintf("date:%d", params.Year))
	}

	if len(queryParts) == 0 {
		// Return empty results if no search parameters
		return []*music.Track{}, nil
	}

	query := strings.Join(queryParts, " AND")

	// Build URL
	baseURL := "https://musicbrainz.org/ws/2/recording"
	searchURL := fmt.Sprintf("%s?query=%s&fmt=json&limit=10", baseURL, url.QueryEscape(query))

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent as required by MusicBrainz
	req.Header.Set("User-Agent", "SoulSolid/1.0 (https://github.com/sst/opencode)")

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp mbRecordingSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to music.Track
	var tracks []*music.Track
	for _, recording := range searchResp.Recordings {
		track := p.convertMBRecordingToTrack(recording)
		if track != nil {
			tracks = append(tracks, track)
		}
	}

	return tracks, nil
}

// convertMBRecordingToTrack converts a MusicBrainz recording to a music.Track
func (p *MusicBrainzProvider) convertMBRecordingToTrack(recording mbRecording) *music.Track {
	// Create artists from artist-credit
	var artists []music.ArtistRole
	for _, credit := range recording.ArtistCredit {
		artist := &music.Artist{Name: credit.Name}
		artists = append(artists, music.ArtistRole{
			Artist: artist,
			Role:   "main",
		})
	}

	// If no artist-credit, create a fallback
	if len(artists) == 0 {
		artists = []music.ArtistRole{
			{Artist: &music.Artist{Name: "Unknown Artist"}, Role: "main"},
		}
	}

	// Find the most relevant release (prefer the one with the earliest date)
	var selectedRelease *mbRelease
	if len(recording.Releases) > 0 {
		selectedRelease = &recording.Releases[0]
		for _, release := range recording.Releases {
			if release.Date != "" && (selectedRelease.Date == "" || release.Date < selectedRelease.Date) {
				selectedRelease = &release
			}
		}
	}

	// Create album
	var album *music.Album
	var year int
	if selectedRelease != nil {
		album = &music.Album{
			Title:   selectedRelease.Title,
			Artists: artists, // Use the same artists for the album
		}

		// Parse year from date
		if selectedRelease.Date != "" {
			if y, err := strconv.Atoi(selectedRelease.Date[:4]); err == nil {
				year = y
			}
		}
	}

	// Get ISRC if available
	var isrc string
	if len(recording.ISRCs) > 0 {
		isrc = recording.ISRCs[0]
	}

	// Create track
	track := &music.Track{
		Title:   recording.Title,
		Artists: artists,
		Album:   album,
		Metadata: music.Metadata{
			Year: year,
		},
		ISRC: isrc,
		SourceData: music.SourceData{
			Source: "musicbrainz",
			URL:    fmt.Sprintf("https://musicbrainz.org/recording/%s", recording.ID),
		},
	}

	return track
}

// Legacy method for backward compatibility
func (p *MusicBrainzProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *MusicBrainzProvider) Name() string    { return "musicbrainz" }
func (p *MusicBrainzProvider) IsEnabled() bool { return p.enabled }
