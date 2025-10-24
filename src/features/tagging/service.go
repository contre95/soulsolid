package tagging

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
)

// Service provides tag editing functionality
type Service struct {
	tagWriter           downloading.TagWriter
	tagReader           importing.TagReader
	libraryRepo         music.Library
	metadataProviders   []MetadataProvider
	fingerprintProvider importing.FingerprintProvider
	config              *config.Manager
}

// NewService creates a new tag service
func NewService(tagWriter downloading.TagWriter, tagReader importing.TagReader, libraryRepo music.Library, metadataProviders []MetadataProvider, fingerprintProvider importing.FingerprintProvider, config *config.Manager) *Service {
	return &Service{
		tagWriter:           tagWriter,
		tagReader:           tagReader,
		libraryRepo:         libraryRepo,
		metadataProviders:   metadataProviders,
		fingerprintProvider: fingerprintProvider,
		config:              config,
	}
}

// GetTrackForEditing retrieves a track with current tag data for editing
func (s *Service) GetTrackForEditing(ctx context.Context, trackID string) (*music.Track, error) {
	slog.Debug("Getting track for editing", "trackID", trackID)

	// Get track from library
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track from library: %w", err)
	}
	if track == nil {
		return nil, fmt.Errorf("track not found: %s", trackID)
	}

	// Read current tags from file to ensure we have the latest data
	currentTrack, err := s.tagReader.ReadFileTags(ctx, track.Path)
	if err != nil {
		slog.Warn("Failed to read current tags from file, using library data", "error", err)
		return track, nil
	}

	// Debug: Check if lyrics were read from file
	if currentTrack.Metadata.Lyrics != "" {
		slog.Debug("Lyrics found in file", "trackID", trackID, "lyricsLength", len(currentTrack.Metadata.Lyrics))
	} else {
		slog.Debug("No lyrics found in file", "trackID", trackID)
	}

	// Merge file data with database relationships
	// Keep database relationships (artists, album) but update metadata from file
	result := *track                                // Copy the database track
	result.Metadata = currentTrack.Metadata         // Use metadata from file
	result.Title = currentTrack.Title               // Use title from file
	result.ISRC = currentTrack.ISRC                 // Use ISRC from file
	result.TitleVersion = currentTrack.TitleVersion // Use title version from file

	// Preserve file-specific data
	result.Path = currentTrack.Path
	result.Format = currentTrack.Format
	result.SampleRate = currentTrack.SampleRate
	result.BitDepth = currentTrack.BitDepth
	result.Channels = currentTrack.Channels
	result.Bitrate = currentTrack.Bitrate

	// Ensure track artists have IDs by matching with database
	result = *s.matchArtistsWithDatabase(ctx, &result)

	return &result, nil
}

// UpdateTrackTags updates the tags of a track file and database
func (s *Service) UpdateTrackTags(ctx context.Context, trackID string, formData map[string]string) error {
	slog.Info("Updating track tags", "trackID", trackID)

	// Get current track data
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Build updated track from form data
	updatedTrack, err := s.buildTrackFromFormData(ctx, track, formData)
	if err != nil {
		return fmt.Errorf("failed to build track from form data: %w", err)
	}

	// Preserve essential fields
	updatedTrack.ID = track.ID
	updatedTrack.AddedDate = track.AddedDate
	updatedTrack.ModifiedDate = time.Now()

	// Handle album creation if needed
	if updatedTrack.Album != nil && updatedTrack.Album.ID == "" {
		// Generate ID for new album
		updatedTrack.Album.ID = uuid.New().String()
		updatedTrack.Album.AddedDate = time.Now()
		updatedTrack.Album.ModifiedDate = time.Now()

		// Add the album to the database
		err = s.libraryRepo.AddAlbum(ctx, updatedTrack.Album)
		if err != nil {
			slog.Error("Failed to add album to database", "albumTitle", updatedTrack.Album.Title, "error", err)
			return fmt.Errorf("failed to add album to database: %w", err)
		}
		slog.Info("Created new album in database", "albumID", updatedTrack.Album.ID, "title", updatedTrack.Album.Title)
	}

	// Write tags to file
	err = s.tagWriter.WriteFileTags(ctx, track.Path, updatedTrack)
	if err != nil {
		return fmt.Errorf("failed to write tags to file: %w", err)
	}

	// Update the album in the database if it exists and has an ID
	if updatedTrack.Album != nil && updatedTrack.Album.ID != "" {
		err = s.libraryRepo.UpdateAlbum(ctx, updatedTrack.Album)
		if err != nil {
			slog.Error("Failed to update album in database", "albumID", updatedTrack.Album.ID, "error", err)
			return fmt.Errorf("failed to update album in database: %w", err)
		}
	}

	// Update the track in the database
	err = s.libraryRepo.UpdateTrack(ctx, updatedTrack)
	if err != nil {
		slog.Error("Failed to update track in database", "trackID", trackID, "error", err)
		return fmt.Errorf("failed to update track in database: %w", err)
	}

	slog.Info("Successfully updated track tags and database", "trackID", trackID, "path", track.Path)
	return nil
}

// buildTrackFromFormData builds a Track struct from form data
func (s *Service) buildTrackFromFormData(ctx context.Context, originalTrack *music.Track, formData map[string]string) (*music.Track, error) {
	track := &music.Track{
		ID:      originalTrack.ID, // Preserve track ID
		Path:    originalTrack.Path,
		Title:   formData["title"],
		Album:   nil, // Will be set below if album is selected
		Artists: make([]music.ArtistRole, 0),
		Metadata: music.Metadata{
			Duration:    originalTrack.Metadata.Duration, // Preserve original duration
			Year:        parseInt(formData["year"]),
			Genre:       formData["genre"],
			TrackNumber: parseInt(formData["track_number"]),
			DiscNumber:  parseInt(formData["disc_number"]),
			Composer:    formData["composer"],
			Lyrics:      formData["lyrics"],
			BPM:         parseFloat(formData["bpm"]),
			Gain:        parseFloat(formData["gain"]),
		},
		ISRC:         formData["isrc"],
		TitleVersion: formData["title_version"],
		MetadataSource: music.MetadataSource{
			Source:            formData["source"],
			MetadataSourceURL: formData["source_url"],
		},
	}

	// Handle artists - support both existing and temporary IDs
	if artistIDsStr := strings.TrimSpace(formData["artist_ids"]); artistIDsStr != "" {
		artistIDs := strings.Split(artistIDsStr, ",")
		for _, artistID := range artistIDs {
			artistID = strings.TrimSpace(artistID)
			if artistID != "" {
				var artist *music.Artist
				var err error

				if strings.HasPrefix(artistID, "temp_") {
					// Handle temporary ID - validate artist exists by name
					artistName := strings.TrimPrefix(artistID, "temp_")
					// First check if artist exists by name (without creating)
					existingArtist, err := s.libraryRepo.GetArtistByName(ctx, artistName)
					if err != nil {
						return nil, fmt.Errorf("error checking if artist '%s' exists: %w", artistName, err)
					}
					if existingArtist == nil {
						return nil, fmt.Errorf("artist '%s' does not exist in library. Please import tracks with this artist first", artistName)
					}
					artist = existingArtist
				} else {
					// Handle existing artist ID - validate it exists
					artist, err = s.libraryRepo.GetArtist(ctx, artistID)
					if err != nil {
						return nil, fmt.Errorf("artist with ID '%s' not found in library: %w", artistID, err)
					}
					if artist == nil {
						return nil, fmt.Errorf("artist with ID '%s' not found in library", artistID)
					}
				}

				track.Artists = append(track.Artists, music.ArtistRole{
					Artist: artist,
					Role:   "main",
				})
			}
		}
	}

	// Handle album - strict dropdown selection only
	if albumID := strings.TrimSpace(formData["album_id"]); albumID != "" {
		album, err := s.libraryRepo.GetAlbum(ctx, albumID)
		if err != nil {
			slog.Warn("Failed to get album by ID, album not updated", "albumID", albumID, "error", err)
			// Keep existing album if lookup fails
		} else {
			track.Album = album
		}
	}

	// Handle album artist - support both existing and temporary IDs
	if albumArtistID := strings.TrimSpace(formData["album_artist_id"]); albumArtistID != "" {
		var albumArtist *music.Artist
		var err error

		if after, ok := strings.CutPrefix(albumArtistID, "temp_"); ok {
			// Handle temporary ID - lookup by name
			artistName := after
			albumArtist, err = s.libraryRepo.FindOrCreateArtist(ctx, artistName)
			if err != nil {
				slog.Warn("Failed to find/create album artist by name, album artist not updated", "artistName", artistName, "error", err)
				// Keep existing album artist if lookup fails
				albumArtist = nil
			}
		} else {
			// Handle existing artist ID
			albumArtist, err = s.libraryRepo.GetArtist(ctx, albumArtistID)
			if err != nil {
				slog.Warn("Failed to get album artist by ID, album artist not updated", "albumArtistID", albumArtistID, "error", err)
				// Keep existing album artist if lookup fails
				albumArtist = nil
			}
		}

		if albumArtist != nil {
			// If no album is selected, create a new album with this artist
			if track.Album == nil {
				albumTitle := "Unknown Album"
				if originalTrack.Album != nil && originalTrack.Album.Title != "" {
					albumTitle = originalTrack.Album.Title
				}

				track.Album = &music.Album{
					Title: albumTitle,
					Artists: []music.ArtistRole{
						{
							Artist: albumArtist,
							Role:   "main",
						},
					},
				}
			} else {
				// Album exists, replace its artists with the selected album artist
				track.Album.Artists = []music.ArtistRole{
					{
						Artist: albumArtist,
						Role:   "main",
					},
				}
			}
		}
	}

	// Preserve format and other metadata
	track.Format = originalTrack.Format
	track.SampleRate = originalTrack.SampleRate
	track.BitDepth = originalTrack.BitDepth
	track.Channels = originalTrack.Channels
	track.Bitrate = originalTrack.Bitrate

	return track, nil
}

// Helper functions for parsing form values
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return 0
}

// matchArtistsWithDatabase tries to match artists in the track with database artists by name
func (s *Service) matchArtistsWithDatabase(ctx context.Context, track *music.Track) *music.Track {
	// Get all artists from database for matching
	dbArtists, err := s.libraryRepo.GetArtists(ctx)
	if err != nil {
		slog.Warn("Failed to get artists for matching", "error", err)
		return track
	}

	// Create a map for quick lookup
	artistMap := make(map[string]*music.Artist)
	for _, artist := range dbArtists {
		artistMap[artist.Name] = artist
	}

	// Match track artists
	for i, artistRole := range track.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID == "" {
			if dbArtist, exists := artistMap[artistRole.Artist.Name]; exists {
				track.Artists[i].Artist = dbArtist
			}
		}
	}

	// Match album artists
	if track.Album != nil {
		for i, artistRole := range track.Album.Artists {
			if artistRole.Artist != nil && artistRole.Artist.ID == "" {
				if dbArtist, exists := artistMap[artistRole.Artist.Name]; exists {
					track.Album.Artists[i].Artist = dbArtist
				}
			}
		}
	}

	return track
}

// GetEnabledProviders returns a map of enabled metadata providers
func (s *Service) GetEnabledProviders() map[string]bool {
	enabled := make(map[string]bool)
	cfg := s.config.Get()
	if cfg.Metadata.Providers != nil {
		for name, provider := range cfg.Metadata.Providers {
			enabled[name] = provider.Enabled
		}
	}
	return enabled
}

// SearchTracksForTrack searches for tracks using current track metadata as search parameters
func (s *Service) SearchTracksForTrack(ctx context.Context, trackID string, providerName string) ([]*music.Track, error) {
	track, err := s.libraryRepo.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return nil, fmt.Errorf("track not found: %s", trackID)
	}

	// Build search parameters from current track data
	searchParams := SearchParams{
		TrackID: track.ID,
		Title:   track.Title,
		Year:    track.Metadata.Year,
	}

	// Add album and album artist if available
	if track.Album != nil {
		searchParams.Album = track.Album.Title
		if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
		}
	}

	// Find the specific provider
	var targetProvider MetadataProvider
	for _, provider := range s.metadataProviders {
		if provider.Name() == providerName && provider.IsEnabled() {
			targetProvider = provider
			break
		}
	}
	if targetProvider == nil {
		return nil, fmt.Errorf("provider '%s' not found or not enabled", providerName)
	}

	// Search for tracks
	tracks, err := targetProvider.SearchTracks(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}

	// Set source data for tracks from this provider
	for _, resultTrack := range tracks {
		// Set source to provider name (if not already set by provider)
		if resultTrack.MetadataSource.Source == "" {
			resultTrack.MetadataSource.Source = providerName
		}
		// Note: URLs should be provided by the metadata providers themselves
		// No URL generation here - providers must provide complete URLs
	}

	// Try to match artists with database artists by name for each result
	for i, resultTrack := range tracks {
		tracks[i] = s.matchArtistsWithDatabase(ctx, resultTrack)
	}

	return tracks, nil
}

// FetchMetadataForTrack fetches metadata for a track using its fingerprint (legacy method for backward compatibility)
func (s *Service) FetchMetadataForTrack(ctx context.Context, trackID string) (*music.Track, error) {
	// For legacy compatibility, use the first enabled provider with search
	for _, provider := range s.metadataProviders {
		if provider.IsEnabled() {
			tracks, err := s.SearchTracksForTrack(ctx, trackID, provider.Name())
			if err == nil && len(tracks) > 0 {
				return tracks[0], nil
			}
		}
	}
	return nil, fmt.Errorf("no metadata found from enabled providers")
}
