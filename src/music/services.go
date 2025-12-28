package music

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LyricsSearchParams contains parameters for searching lyrics
type LyricsSearchParams struct {
	TrackID     string
	AlbumArtist string
	Album       string
	Title       string
	Artist      string
}

// LyricsProviderInfo contains information about a lyrics provider for the UI
type LyricsProviderInfo struct {
	Name        string
	DisplayName string
	Enabled     bool
}

// MetadataService defines the interface for tagging operations
type MetadataService interface {
	AddChromaprintAndAcoustID(ctx context.Context, trackID string) error
	GetTrackFileTags(ctx context.Context, trackID string) (*Track, error)
	UpdateTrackTags(ctx context.Context, trackID string, formData map[string]string) error
}

// LyricsService defines the interface for lyrics operations
type LyricsService interface {
	AddLyrics(ctx context.Context, trackID string, providerName string) error
	GetEnabledLyricsProviders() map[string]bool
	GetLyricsProvidersInfo() []LyricsProviderInfo
	SearchLyrics(ctx context.Context, trackID string, providerName string) (string, error)
}

// JobService defines the interface for job management
type JobService interface {
	StartJob(jobType string, name string, metadata map[string]any) (string, error)
	UpdateJobProgress(jobID string, progress int, message string)
}

// generateID creates a new UUID string
func generateID() string {
	return uuid.New().String()
}

// PlaylistService defines the interface for playlist operations
type PlaylistService interface {
	CreatePlaylist(ctx context.Context, name, description string) (*Playlist, error)
	GetPlaylist(ctx context.Context, id string) (*Playlist, error)
	UpdatePlaylist(ctx context.Context, playlist *Playlist) error
	DeletePlaylist(ctx context.Context, id string) error
	GetPlaylists(ctx context.Context) ([]*Playlist, error)
	AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) error
	RemoveTrackFromPlaylist(ctx context.Context, playlistID, trackID string) error
	GetPlaylistTracks(ctx context.Context, playlistID string) ([]*Track, error)
}

// DefaultPlaylistService implements PlaylistService
type DefaultPlaylistService struct {
	library Library
}

// NewPlaylistService creates a new playlist service
func NewPlaylistService(library Library) PlaylistService {
	return &DefaultPlaylistService{
		library: library,
	}
}

// CreatePlaylist creates a new playlist
func (s *DefaultPlaylistService) CreatePlaylist(ctx context.Context, name, description string) (*Playlist, error) {
	now := time.Now()
	playlist := &Playlist{
		ID:           generateID(),
		Name:         name,
		Description:  description,
		CreatedDate:  now,
		ModifiedDate: now,
		Attributes:   make(map[string]string),
	}

	err := s.library.AddPlaylist(ctx, playlist)
	if err != nil {
		return nil, err
	}

	return playlist, nil
}

// GetPlaylist retrieves a playlist by ID
func (s *DefaultPlaylistService) GetPlaylist(ctx context.Context, id string) (*Playlist, error) {
	playlist, err := s.library.GetPlaylist(ctx, id)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, nil
	}

	// Load tracks
	tracks, err := s.library.GetPlaylistTracks(ctx, id)
	if err != nil {
		return nil, err
	}
	playlist.Tracks = tracks

	return playlist, nil
}

// UpdatePlaylist updates a playlist
func (s *DefaultPlaylistService) UpdatePlaylist(ctx context.Context, playlist *Playlist) error {
	playlist.ModifiedDate = time.Now()
	return s.library.UpdatePlaylist(ctx, playlist)
}

// DeletePlaylist deletes a playlist
func (s *DefaultPlaylistService) DeletePlaylist(ctx context.Context, id string) error {
	return s.library.DeletePlaylist(ctx, id)
}

// GetPlaylists retrieves all playlists
func (s *DefaultPlaylistService) GetPlaylists(ctx context.Context) ([]*Playlist, error) {
	return s.library.GetPlaylists(ctx)
}

// AddTrackToPlaylist adds a track to a playlist
func (s *DefaultPlaylistService) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) error {
	return s.library.AddTrackToPlaylist(ctx, playlistID, trackID)
}

// RemoveTrackFromPlaylist removes a track from a playlist
func (s *DefaultPlaylistService) RemoveTrackFromPlaylist(ctx context.Context, playlistID, trackID string) error {
	return s.library.RemoveTrackFromPlaylist(ctx, playlistID, trackID)
}

// GetPlaylistTracks retrieves tracks in a playlist
func (s *DefaultPlaylistService) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*Track, error) {
	return s.library.GetPlaylistTracks(ctx, playlistID)
}
