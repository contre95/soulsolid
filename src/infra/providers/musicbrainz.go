package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
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

type mbReleaseSearchResponse struct {
	Releases []mbReleaseResult `json:"releases"`
}

// mbReleaseResult is one hit of a release search. track-count is the total across all media.
type mbReleaseResult struct {
	ID           string           `json:"id"`
	Title        string           `json:"title"`
	Status       string           `json:"status"`
	TrackCount   int              `json:"track-count"`
	ArtistCredit []mbArtistCredit `json:"artist-credit"`
}

// MusicBrainzProvider implements MetadataProvider for MusicBrainz
type MusicBrainzProvider struct {
	enabled bool
}

// NewMusicBrainzProvider creates a new MusicBrainz provider
func NewMusicBrainzProvider(enabled bool) *MusicBrainzProvider {
	return &MusicBrainzProvider{enabled: enabled}
}

func (p *MusicBrainzProvider) SearchTracks(ctx context.Context, params metadata.SearchParams) ([]*music.Track, error) {
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
	if params.AcoustID != "" {
		queryParts = append(queryParts, fmt.Sprintf("recording_acoustid:\"%s\"", params.AcoustID))
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
	resp, err := httpClient.Do(req)
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

var _ metadata.AlbumMetadataProvider = (*MusicBrainzProvider)(nil)

// FetchAlbumInfo looks an album up on MusicBrainz and returns its official track count.
//
// A release group ID or barcode, when known, pins the lookup; otherwise the release is searched
// by title and artist and only exact (case and punctuation insensitive) matches are kept.
// A release group holds many editions, so among the matching releases (official ones if there
// are any) the most common track count wins, the smaller one on a tie.
func (p *MusicBrainzProvider) FetchAlbumInfo(ctx context.Context, params metadata.AlbumSearchParams) (*metadata.AlbumInfo, error) {
	var query string
	byTitle := false
	switch {
	case isUUID(params.ReleaseGroupID):
		query = fmt.Sprintf("rgid:%s", params.ReleaseGroupID)
	case isDigits(params.Barcode):
		query = fmt.Sprintf("barcode:%s", params.Barcode)
	case strings.TrimSpace(params.Album) != "":
		byTitle = true
		query = fmt.Sprintf("release:\"%s\"", stripQuotes(params.Album))
		if strings.TrimSpace(params.AlbumArtist) != "" {
			query += fmt.Sprintf(" AND artist:\"%s\"", stripQuotes(params.AlbumArtist))
		}
	default:
		return nil, nil
	}

	searchURL := fmt.Sprintf("https://musicbrainz.org/ws/2/release?query=%s&fmt=json&limit=25", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "SoulSolid/1.0 (https://github.com/contre95/soulsolid)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MusicBrainz API request failed with status %d", resp.StatusCode)
	}

	var searchResp mbReleaseSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	wantTitle := normalizeMatch(params.Album)
	wantArtist := normalizeMatch(params.AlbumArtist)
	var candidates []mbReleaseResult
	for _, r := range searchResp.Releases {
		if r.TrackCount <= 0 {
			continue
		}
		if byTitle {
			if normalizeMatch(r.Title) != wantTitle {
				continue
			}
			if wantArtist != "" && !mbCreditMatches(r.ArtistCredit, wantArtist) {
				continue
			}
		}
		candidates = append(candidates, r)
	}
	best, ok := pickMBRelease(candidates)
	if !ok {
		return nil, nil
	}

	artist := ""
	if len(best.ArtistCredit) > 0 {
		artist = best.ArtistCredit[0].Name
	}
	return &metadata.AlbumInfo{
		Title:       best.Title,
		Artist:      artist,
		TotalTracks: best.TrackCount,
		Source:      "musicbrainz",
		SourceURL:   fmt.Sprintf("https://musicbrainz.org/release/%s", best.ID),
	}, nil
}

// pickMBRelease chooses the release whose track count represents the album: among the official
// releases (or all of them when none is official) the most common track count wins, the smaller
// one on a tie. Releases come ordered by relevance, so the first one with that count is returned.
func pickMBRelease(candidates []mbReleaseResult) (mbReleaseResult, bool) {
	if official := filterOfficial(candidates); len(official) > 0 {
		candidates = official
	}
	if len(candidates) == 0 {
		return mbReleaseResult{}, false
	}
	freq := map[int]int{}
	for _, r := range candidates {
		freq[r.TrackCount]++
	}
	best := candidates[0]
	for _, r := range candidates {
		f, bf := freq[r.TrackCount], freq[best.TrackCount]
		if f > bf || (f == bf && r.TrackCount < best.TrackCount) {
			best = r
		}
	}
	return best, true
}

func mbCreditMatches(credits []mbArtistCredit, wantArtist string) bool {
	for _, c := range credits {
		if normalizeMatch(c.Name) == wantArtist || normalizeMatch(c.Artist.Name) == wantArtist {
			return true
		}
	}
	return false
}

func filterOfficial(releases []mbReleaseResult) []mbReleaseResult {
	var official []mbReleaseResult
	for _, r := range releases {
		if strings.EqualFold(r.Status, "Official") {
			official = append(official, r)
		}
	}
	return official
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return s != "" && err == nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
		MetadataSource: music.MetadataSource{
			Source:            "musicbrainz",
			MetadataSourceURL: fmt.Sprintf("https://musicbrainz.org/recording/%s", recording.ID),
		},
		HasLyrics: true,
	}

	return track
}

// Legacy method for backward compatibility
func (p *MusicBrainzProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	tracks, err := p.SearchTracks(ctx, metadata.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *MusicBrainzProvider) Name() string    { return "musicbrainz" }
func (p *MusicBrainzProvider) IsEnabled() bool { return p.enabled }
