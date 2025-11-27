package lyrics

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// LyricsResult represents a lyrics result from a provider with quality score
type LyricsResult struct {
	Lyrics     string
	Provider   string
	Quality    int
	Confidence float64
}

// Service provides lyrics functionality
type Service struct {
	tagWriter       TagWriter
	tagReader       TagReader
	libraryRepo     music.Library
	lyricsProviders []LyricsProvider
	config          *config.Manager
}

// TagWriter interface for writing tags
type TagWriter interface {
	WriteFileTags(ctx context.Context, path string, track *music.Track) error
}

// TagReader interface for reading tags
type TagReader interface {
	ReadFileTags(ctx context.Context, path string) (*music.Track, error)
}

// NewService creates a new lyrics service
func NewService(tagWriter TagWriter, tagReader TagReader, libraryRepo music.Library, lyricsProviders []LyricsProvider, config *config.Manager) *Service {
	return &Service{
		tagWriter:       tagWriter,
		tagReader:       tagReader,
		libraryRepo:     libraryRepo,
		lyricsProviders: lyricsProviders,
		config:          config,
	}
}

// SetLyricsToNoLyrics sets the lyrics to "[No Lyrics]" for a track
func (s *Service) SetLyricsToNoLyrics(ctx context.Context, trackID string) error {
	// Get current track data
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Set lyrics to [No Lyrics]
	track.Metadata.Lyrics = "[No Lyrics]"
	track.ModifiedDate = time.Now()

	// Write lyrics to file tags
	if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
		slog.Warn("Failed to write [No Lyrics] to file tags", "error", err, "trackID", trackID, "path", track.Path)
		// Continue - we still want to update the database
	}

	// Update track in database
	err = s.libraryRepo.UpdateTrack(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track with [No Lyrics]: %w", err)
	}

	slog.Info("Successfully set [No Lyrics] for track", "trackID", trackID)
	return nil
}

// AddLyrics searches for and adds lyrics to a track using a specific provider
func (s *Service) AddLyrics(ctx context.Context, trackID string, providerName string) error {
	// Get current track data
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Skip if lyrics already exist
	if track.Metadata.Lyrics != "" {
		slog.Debug("Track already has lyrics", "trackID", trackID)
		return nil
	}

	// Build search parameters from current track data
	searchParams := music.LyricsSearchParams{
		TrackID: track.ID,
		Title:   track.Title,
	}

	// Add artist if available
	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		searchParams.Artist = track.Artists[0].Artist.Name
	}

	// Add album and album artist if available
	if track.Album != nil {
		searchParams.Album = track.Album.Title
		if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
		}
	}

	// Find the specific provider
	var targetProvider LyricsProvider
	for _, provider := range s.lyricsProviders {
		if provider.Name() == providerName && provider.IsEnabled() {
			targetProvider = provider
			break
		}
	}
	if targetProvider == nil {
		return fmt.Errorf("lyrics provider '%s' not found or not enabled", providerName)
	}

	// Search for lyrics using the specified provider
	slog.Debug("Trying lyrics provider", "provider", targetProvider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist)
	lyrics, err := targetProvider.SearchLyrics(ctx, searchParams)
	if err != nil {
		slog.Warn("Failed to search lyrics with provider", "provider", targetProvider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist, "error", err.Error())
		return fmt.Errorf("failed to search lyrics with provider '%s': %w", providerName, err)
	}

	if lyrics == "" {
		slog.Info("No lyrics found for track with provider", "trackID", trackID, "title", track.Title, "artist", searchParams.Artist, "provider", providerName)
		return nil // Not an error if no lyrics found, just return
	}

	quality, confidence := s.scoreLyrics(lyrics, searchParams)
	slog.Info("Found lyrics with provider", "provider", targetProvider.Name(), "trackID", trackID, "quality", quality, "confidence", confidence, "lyricsLength", len(lyrics))

	preview := lyrics
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	slog.Info("Adding lyrics for track", "provider", providerName, "trackID", trackID, "quality", quality, "confidence", confidence, "lyricsLength", len(lyrics), "lyricsPreview", preview)

	// Update the track with the lyrics
	track.Metadata.Lyrics = lyrics
	track.ModifiedDate = time.Now()

	// Write lyrics to file tags
	if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
		slog.Warn("Failed to write lyrics to file tags", "error", err, "trackID", trackID, "path", track.Path)
		// Continue - we still want to update the database
	}

	// Update track in database
	err = s.libraryRepo.UpdateTrack(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track with lyrics: %w", err)
	}

	slog.Info("Successfully added lyrics for track", "trackID", trackID, "provider", providerName, "quality", quality, "lyricsLength", len(lyrics))
	return nil
}

// GetEnabledLyricsProviders returns a map of enabled lyrics providers
func (s *Service) GetEnabledLyricsProviders() map[string]bool {
	return s.config.GetEnabledLyricsProviders()
}

// SearchLyrics searches for lyrics using a given track and lyrics provider
func (s *Service) SearchLyrics(ctx context.Context, trackID string, providerName string) (string, error) {
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return "", fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return "", fmt.Errorf("track not found: %s", trackID)
	}

	// Build search parameters from current track data
	searchParams := music.LyricsSearchParams{
		TrackID: track.ID,
		Title:   track.Title,
	}

	// Add artist if available
	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		searchParams.Artist = track.Artists[0].Artist.Name
	}

	// Add album and album artist if available
	if track.Album != nil {
		searchParams.Album = track.Album.Title
		if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
		}
	}

	// Find the specific provider
	var targetProvider LyricsProvider
	for _, provider := range s.lyricsProviders {
		if provider.Name() == providerName && provider.IsEnabled() {
			targetProvider = provider
			break
		}
	}
	if targetProvider == nil {
		return "", fmt.Errorf("lyrics provider '%s' not found or not enabled", providerName)
	}

	// Search for lyrics
	lyrics, err := targetProvider.SearchLyrics(ctx, searchParams)
	if err != nil {
		return "", fmt.Errorf("failed to search lyrics: %w", err)
	}

	return lyrics, nil
}

// AddLyricsWithBestProvider searches for and adds lyrics to a track by trying all enabled providers and selecting the best quality
// This method maintains the original behavior for jobs and bulk operations
func (s *Service) AddLyricsWithBestProvider(ctx context.Context, trackID string) error {
	// Get current track data
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Skip if lyrics already exist
	if track.Metadata.Lyrics != "" {
		slog.Debug("Track already has lyrics", "trackID", trackID)
		return nil
	}

	// Build search parameters from current track data
	searchParams := music.LyricsSearchParams{
		TrackID: track.ID,
		Title:   track.Title,
	}

	// Add artist if available
	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		searchParams.Artist = track.Artists[0].Artist.Name
	}

	// Add album and album artist if available
	if track.Album != nil {
		searchParams.Album = track.Album.Title
		if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
		}
	}

	// Collect lyrics from all enabled providers and select best
	var results []*LyricsResult
	for _, provider := range s.lyricsProviders {
		if !provider.IsEnabled() {
			continue
		}

		slog.Debug("Trying lyrics provider", "provider", provider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist)
		lyrics, err := provider.SearchLyrics(ctx, searchParams)
		if err != nil {
			slog.Warn("Failed to search lyrics with provider", "provider", provider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist, "error", err.Error())
			continue
		}

		if lyrics != "" {
			quality, confidence := s.scoreLyrics(lyrics, searchParams)
			result := &LyricsResult{
				Lyrics:     lyrics,
				Provider:   provider.Name(),
				Quality:    quality,
				Confidence: confidence,
			}
			results = append(results, result)

			slog.Info("Found lyrics with provider", "provider", provider.Name(), "trackID", trackID, "quality", quality, "confidence", confidence, "lyricsLength", len(lyrics))
		}
	}

	if len(results) == 0 {
		slog.Info("No lyrics found for track with any provider", "trackID", trackID, "title", track.Title, "artist", searchParams.Artist, "providers", len(s.lyricsProviders))
		return nil // Not an error if no lyrics found, just return
	}

	// Select the result with the highest quality score
	bestResult := results[0]
	for _, result := range results[1:] {
		if result.Quality > bestResult.Quality ||
			(result.Quality == bestResult.Quality && result.Confidence > bestResult.Confidence) {
			bestResult = result
		}
	}

	preview := bestResult.Lyrics
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	slog.Info("Selected best lyrics", "provider", bestResult.Provider, "trackID", trackID, "quality", bestResult.Quality, "confidence", bestResult.Confidence, "lyricsLength", len(bestResult.Lyrics), "lyricsPreview", preview)

	// Update the track with the best lyrics
	track.Metadata.Lyrics = bestResult.Lyrics
	track.ModifiedDate = time.Now()

	// Write lyrics to file tags
	if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
		slog.Warn("Failed to write lyrics to file tags", "error", err, "trackID", trackID, "path", track.Path)
		// Continue - we still want to update the database
	}

	// Update track in database
	err = s.libraryRepo.UpdateTrack(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track with lyrics: %w", err)
	}

	slog.Info("Successfully added lyrics for track", "trackID", trackID, "provider", bestResult.Provider, "quality", bestResult.Quality, "lyricsLength", len(bestResult.Lyrics))
	return nil
}

// scoreLyrics evaluates the quality of lyrics based on various factors
func (s *Service) scoreLyrics(lyrics string, params music.LyricsSearchParams) (int, float64) {
	quality := 0
	confidence := 0.0

	// Basic quality checks
	if lyrics == "" {
		return 0, 0.0
	}

	// Length scoring - prefer substantial lyrics
	length := len(strings.TrimSpace(lyrics))
	if length > 500 {
		quality += 30
		confidence += 0.3
	} else if length > 200 {
		quality += 20
		confidence += 0.2
	} else if length > 50 {
		quality += 10
		confidence += 0.1
	}

	// Structure scoring - look for verse/chorus patterns
	lines := strings.Split(lyrics, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines > 8 {
		quality += 20
		confidence += 0.2
	} else if nonEmptyLines > 4 {
		quality += 10
		confidence += 0.1
	}

	// Content quality scoring
	lowerLyrics := strings.ToLower(lyrics)

	// Penalize common non-lyric content
	nonLyricPatterns := []string{
		`lyrics? by`,
		`copyright`,
		`all rights reserved`,
		`buy.*track`,
		`download.*now`,
		`instrumental`,
		`[music]`,
		`[intro]`,
		`[outro]`,
	}

	for _, pattern := range nonLyricPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerLyrics); matched {
			quality -= 10
			confidence -= 0.1
		}
	}

	// Reward good lyrical content
	goodPatterns := []string{
		`\[verse\]`,
		`\[chorus\]`,
		`\[bridge\]`,
		`\[pre-chorus\]`,
	}

	for _, pattern := range goodPatterns {
		if matched, _ := regexp.MatchString(pattern, lowerLyrics); matched {
			quality += 15
			confidence += 0.15
		}
	}

	// Title/artist matching bonus
	if params.Title != "" {
		titleLower := strings.ToLower(params.Title)
		if strings.Contains(lowerLyrics, titleLower) {
			quality += 10
			confidence += 0.1
		}
	}

	if params.Artist != "" {
		artistLower := strings.ToLower(params.Artist)
		if strings.Contains(lowerLyrics, artistLower) {
			quality += 5
			confidence += 0.05
		}
	}

	// Language detection bonus (basic check for English lyrics)
	englishWords := regexp.MustCompile(`\b(the|and|you|love|i'm|can't|will|just|know|like|have|this|that|with|from|they|been|when|make|what|some|could|them|than|then|back|only|take|time|come|here|more|want|look|your|was|way|out|well|now|say|she|her|his|him|how|its|our|their|who|did|get|has|him|his|its|may|new|now|old|see|she|try|two|use|way|who|why|did)\b`)
	englishMatches := englishWords.FindAllString(lowerLyrics, -1)
	if len(englishMatches) > 10 {
		quality += 10
		confidence += 0.1
	}

	// Penalty for repetitive content
	words := strings.Fields(lowerLyrics)
	if len(words) > 0 {
		uniqueWords := make(map[string]bool)
		for _, word := range words {
			uniqueWords[word] = true
		}
		uniquenessRatio := float64(len(uniqueWords)) / float64(len(words))
		if uniquenessRatio < 0.3 {
			quality -= 15
			confidence -= 0.15
		} else if uniquenessRatio > 0.7 {
			quality += 10
			confidence += 0.1
		}
	}

	// Ensure quality doesn't go below 0 and confidence is between 0 and 1
	if quality < 0 {
		quality = 0
	}
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	return quality, confidence
}
