package lyrics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service provides lyrics functionality
type Service struct {
	tagWriter       TagWriter
	tagReader       TagReader
	libraryRepo     music.Library
	lyricsProviders map[string]LyricsProvider
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
func NewService(tagWriter TagWriter, tagReader TagReader, libraryRepo music.Library, lyricsProviders map[string]LyricsProvider, config *config.Manager) *Service {
	return &Service{
		tagWriter:       tagWriter,
		tagReader:       tagReader,
		libraryRepo:     libraryRepo,
		lyricsProviders: lyricsProviders,
		config:          config,
	}
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
	// NOTE: Do we really need to check for Artists and Album?
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
	targetProvider, exists := s.lyricsProviders[providerName]
	if !exists || targetProvider == nil || !targetProvider.IsEnabled() {
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

	slog.Info("Found lyrics with provider", "provider", targetProvider.Name(), "trackID", trackID, "lyricsLength", len(lyrics))

	preview := lyrics
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	slog.Info("Adding lyrics for track", "provider", providerName, "trackID", trackID, "lyricsLength", len(lyrics), "lyricsPreview", preview)

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

	slog.Info("Successfully added lyrics for track", "trackID", trackID, "provider", providerName, "lyricsLength", len(lyrics))
	return nil
}

// GetEnabledLyricsProviders returns a map of enabled lyrics providers
func (s *Service) GetEnabledLyricsProviders() map[string]bool {
	return s.config.GetEnabledLyricsProviders()
}

// LyricsProviderInfo contains information about a lyrics provider for the UI
// GetLyricsProvidersInfo returns a slice of lyrics provider information for the UI
func (s *Service) GetLyricsProvidersInfo() []music.LyricsProviderInfo {
	var providers []music.LyricsProviderInfo
	enabledProviders := s.config.GetEnabledLyricsProviders()

	for name, provider := range s.lyricsProviders {
		providers = append(providers, music.LyricsProviderInfo{
			Name:        provider.Name(),
			DisplayName: provider.DisplayName(),
			Enabled:     enabledProviders[name] && provider.IsEnabled(),
		})
	}

	return providers
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
	targetProvider, exists := s.lyricsProviders[providerName]
	if !exists || targetProvider == nil || !targetProvider.IsEnabled() {
		return "", fmt.Errorf("lyrics provider '%s' not found or not enabled", providerName)
	}

	// Search for lyrics
	lyrics, err := targetProvider.SearchLyrics(ctx, searchParams)
	if err != nil {
		return "", fmt.Errorf("failed to search lyrics: %w", err)
	}

	return lyrics, nil
}
