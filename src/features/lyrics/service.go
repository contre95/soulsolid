package lyrics

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// AddLyricsResult represents the outcome of an AddLyrics operation
type AddLyricsResult int

const (
	LyricsAdded   AddLyricsResult = iota // Lyrics were added to a track without lyrics
	LyricsQueued                         // Lyrics were queued for existing track (different or failed)
	LyricsSkipped                        // Lyrics were skipped (identical to existing)
)

// Service provides lyrics functionality
type Service struct {
	tagWriter       TagWriter
	tagReader       TagReader
	libraryRepo     music.Library
	lyricsProviders map[string]LyricsProvider
	config          *config.Manager
	queue           music.Queue
	jobService      music.JobService
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
func NewService(tagWriter TagWriter, tagReader TagReader, libraryRepo music.Library, lyricsProviders map[string]LyricsProvider, config *config.Manager, queue music.Queue, jobService music.JobService) *Service {
	return &Service{
		tagWriter:       tagWriter,
		tagReader:       tagReader,
		libraryRepo:     libraryRepo,
		lyricsProviders: lyricsProviders,
		config:          config,
		queue:           queue,
		jobService:      jobService,
	}
}

// AddLyricsQueueItem adds a track to the lyrics queue
func (s *Service) AddLyricsQueueItem(track *music.Track, qType music.QueueItemType, metadata map[string]string) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}
	item := music.QueueItem{
		ID:        track.ID,
		Type:      qType,
		Track:     track,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	return s.queue.Add(item)
}

// GetLyricsQueueItems returns all items in the lyrics queue
func (s *Service) GetLyricsQueueItems() map[string]music.QueueItem {
	return s.queue.GetAll()
}

// ProcessLyricsQueueItem processes a lyrics queue item with the given action
func (s *Service) ProcessLyricsQueueItem(ctx context.Context, itemID string, action string) error {
	item, err := s.queue.GetByID(itemID)
	if err != nil {
		return fmt.Errorf("queue item not found: %w", err)
	}
	if item.Track == nil {
		return fmt.Errorf("queue item does not contain a valid track")
	}

	track := item.Track
	switch item.Type {
	case ExistingLyrics:
		switch action {
		case "override":
			newLyrics, hasNewLyrics := item.Metadata["new_lyrics"]
			if !hasNewLyrics || newLyrics == "" {
				slog.Warn("Override action called but no new lyrics in queue item", "itemID", itemID, "trackID", track.ID)
				return fmt.Errorf("no new lyrics found in queue item")
			}
			slog.Info("Overriding existing lyrics with new lyrics", "trackID", track.ID, "newLength", len(newLyrics), "provider", item.Metadata["provider"])
			track.Metadata.Lyrics = newLyrics
			track.HasLyrics = true
			track.ModifiedDate = time.Now()
			if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
				slog.Warn("Failed to write lyrics to file tags", "error", err, "trackID", track.ID)
			}
			if err := s.libraryRepo.UpdateTrack(ctx, track); err != nil {
				return fmt.Errorf("failed to update track with new lyrics: %w", err)
			}
			slog.Info("Successfully overridden lyrics", "trackID", track.ID, "provider", item.Metadata["provider"])
			return s.queue.Remove(itemID)
		case "keep_old":
			// Just remove from queue
			return s.queue.Remove(itemID)
		default:
			return fmt.Errorf("invalid action '%s' for existing_lyrics, expected 'override' or 'keep_old'", action)
		}
	case Lyric404:
		switch action {
		case "no_lyrics":
			// Clear lyrics field and set HasLyrics=false
			track.Metadata.Lyrics = ""
			track.HasLyrics = false
			track.ModifiedDate = time.Now()
			// Write to file tags (clear lyrics)
			if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
				slog.Warn("Failed to clear lyrics in file tags", "error", err, "trackID", track.ID)
			}
			// Update database
			if err := s.libraryRepo.UpdateTrack(ctx, track); err != nil {
				return fmt.Errorf("failed to update track with cleared lyrics: %w", err)
			}
			slog.Info("Marked track as having no lyrics", "trackID", track.ID)
			return s.queue.Remove(itemID)
		default:
			return fmt.Errorf("invalid action '%s' for lyric_404, expected 'no_lyrics'", action)
		}
	case FailedLyrics:
		switch action {
		case "skip":
			// Remove from queue without changes
			return s.queue.Remove(itemID)
		case "edit_manual":
			// Remove from queue, user will edit manually
			return s.queue.Remove(itemID)
		case "no_lyrics":
			// Clear lyrics field and set HasLyrics=false
			track.Metadata.Lyrics = ""
			track.HasLyrics = false
			track.ModifiedDate = time.Now()
			// Write to file tags (clear lyrics)
			if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
				slog.Warn("Failed to clear lyrics in file tags", "error", err, "trackID", track.ID)
			}
			// Update database
			if err := s.libraryRepo.UpdateTrack(ctx, track); err != nil {
				return fmt.Errorf("failed to update track with cleared lyrics: %w", err)
			}
			slog.Info("Marked track as having no lyrics", "trackID", track.ID)
			return s.queue.Remove(itemID)
		default:
			return fmt.Errorf("invalid action '%s' for failed_lyrics, expected 'skip', 'edit_manual', or 'no_lyrics'", action)
		}
	default:
		return fmt.Errorf("unsupported queue item type: %s", item.Type)
	}
}

// ProcessLyricsQueueGroup processes all items in a group with the given action
func (s *Service) ProcessLyricsQueueGroup(ctx context.Context, groupKey string, groupType string, action string) error {
	// Get the group items first to process them individually
	var groupItems []music.QueueItem

	if groupType == "artist" {
		groups := s.GetLyricsGroupedByArtist()
		groupItems = groups[groupKey]
	} else if groupType == "album" {
		groups := s.GetLyricsGroupedByAlbum()
		groupItems = groups[groupKey]
	} else {
		return fmt.Errorf("invalid group type: %s", groupType)
	}

	if len(groupItems) == 0 {
		return fmt.Errorf("no items found in group %s", groupKey)
	}

	// Validate action based on item types (we could do per-item validation but each item type may have different allowed actions)
	// We'll rely on ProcessLyricsQueueItem to validate per item.

	// Process each item in the group
	for _, item := range groupItems {
		if err := s.ProcessLyricsQueueItem(ctx, item.ID, action); err != nil {
			slog.Warn("Failed to process lyrics queue item in group", "itemID", item.ID, "action", action, "error", err)
			// Continue processing other items even if one fails
		}
	}

	return nil
}

// ClearLyricsQueue removes all items from the lyrics queue
func (s *Service) ClearLyricsQueue() error {
	return s.queue.Clear()
}

// GetLyricsGroupedByArtist returns lyrics queue items grouped by artist
func (s *Service) GetLyricsGroupedByArtist() map[string][]music.QueueItem {
	return s.queue.GetGroupedByArtist()
}

// GetLyricsGroupedByAlbum returns lyrics queue items grouped by album
func (s *Service) GetLyricsGroupedByAlbum() map[string][]music.QueueItem {
	return s.queue.GetGroupedByAlbum()
}

// AddLyrics searches for and adds lyrics to a track using a specific provider
func (s *Service) AddLyrics(ctx context.Context, trackID string, providerName string) (AddLyricsResult, error) {
	// Get current track data
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return LyricsSkipped, fmt.Errorf("failed to get track: %w", err)
	}
	// If lyrics already exist, fetch new lyrics and add to queue for manual decision
	if track.Metadata.Lyrics != "" {
		searchParams := music.LyricsSearchParams{
			TrackID: track.ID,
			Title:   track.Title,
		}
		if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
			searchParams.Artist = track.Artists[0].Artist.Name
		}
		if track.Album != nil {
			searchParams.Album = track.Album.Title
			if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
				searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
			}
		}

		targetProvider, exists := s.lyricsProviders[providerName]
		if !exists || targetProvider == nil || !targetProvider.IsEnabled() {
			return LyricsSkipped, fmt.Errorf("lyrics provider '%s' not found or not enabled", providerName)
		}

		newLyrics, err := targetProvider.SearchLyrics(ctx, searchParams)
		if err != nil {
			slog.Warn("Failed to search new lyrics for existing lyrics queue", "provider", providerName, "trackID", trackID, "error", err)
			return LyricsSkipped, nil
		}

		if newLyrics != "" {
			existingTrimmed := strings.TrimSpace(track.Metadata.Lyrics)
			newTrimmed := strings.TrimSpace(newLyrics)
			slog.Info("Track already has lyrics, comparing with fetched", "trackID", trackID, "existingLength", len(track.Metadata.Lyrics), "newLength", len(newLyrics), "identical", existingTrimmed == newTrimmed)
			if existingTrimmed != newTrimmed {
				slog.Info("Track already has lyrics, new lyrics differ, adding to queue", "trackID", trackID)
				if err := s.AddLyricsQueueItem(track, ExistingLyrics, map[string]string{
					"provider":   providerName,
					"new_lyrics": newLyrics,
				}); err != nil {
					slog.Warn("Failed to add track to lyrics queue", "trackID", trackID, "error", err)
				} else {
					slog.Debug("Successfully added track to existing_lyrics queue with new lyrics", "trackID", trackID)
				}
				return LyricsQueued, nil
			} else {
				slog.Info("Track already has lyrics, new lyrics are identical, skipping queue", "trackID", trackID)
			}
		} else {
			slog.Info("Track already has lyrics, no new lyrics found", "trackID", trackID)
		}
		return LyricsSkipped, nil
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
		return LyricsSkipped, fmt.Errorf("lyrics provider '%s' not found or not enabled", providerName)
	}

	// Search for lyrics using the specified provider
	slog.Debug("Trying lyrics provider", "provider", targetProvider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist)
	lyrics, err := targetProvider.SearchLyrics(ctx, searchParams)
	if err != nil {
		slog.Warn("Failed to search lyrics with provider", "provider", targetProvider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist, "error", err.Error())
		// Add to failed_lyrics queue for manual decision
		if err := s.AddLyricsQueueItem(track, FailedLyrics, map[string]string{"provider": providerName, "error": err.Error()}); err != nil {
			slog.Warn("Failed to add track to lyrics queue", "trackID", trackID, "error", err)
		}
		return LyricsQueued, nil
	}

	if lyrics == "" {
		slog.Info("No lyrics found for track with provider", "trackID", trackID, "title", track.Title, "artist", searchParams.Artist, "provider", providerName)
		// Add to lyric_404 queue for manual decision
		if err := s.AddLyricsQueueItem(track, Lyric404, map[string]string{"provider": providerName}); err != nil {
			slog.Warn("Failed to add track to lyrics queue", "trackID", trackID, "error", err)
		}
		return LyricsQueued, nil
	}

	slog.Info("Found lyrics with provider", "provider", targetProvider.Name(), "trackID", trackID, "lyricsLength", len(lyrics))

	preview := lyrics
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	slog.Info("Adding lyrics for track", "provider", providerName, "trackID", trackID, "lyricsLength", len(lyrics), "lyricsPreview", preview)

	// Update the track with the lyrics
	track.Metadata.Lyrics = lyrics
	track.HasLyrics = true
	track.ModifiedDate = time.Now()

	// Write lyrics to file tags
	if err := s.tagWriter.WriteFileTags(ctx, track.Path, track); err != nil {
		slog.Warn("Failed to write lyrics to file tags", "error", err, "trackID", trackID, "path", track.Path)
		// Continue - we still want to update the database
	}

	// Update track in database
	err = s.libraryRepo.UpdateTrack(ctx, track)
	if err != nil {
		return LyricsSkipped, fmt.Errorf("failed to update track with lyrics: %w", err)
	}

	slog.Info("Successfully added lyrics for track", "trackID", trackID, "provider", providerName, "lyricsLength", len(lyrics))
	return LyricsAdded, nil
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

// StartLyricsAnalysis starts a job to analyze all tracks for lyrics
func (s *Service) StartLyricsAnalysis(ctx context.Context, provider string) (string, error) {
	slog.Info("Starting lyrics analysis job", "provider", provider)
	jobID, err := s.jobService.StartJob("analyze_lyrics", "Analyze Lyrics for Library", map[string]any{
		"provider": provider,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start lyrics analysis job: %w", err)
	}
	slog.Info("Lyrics analysis job started", "jobID", jobID, "provider", provider)
	return jobID, nil
}
