package playlists

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service is the domain service for the playlists feature.
type Service struct {
	playlistRepo  music.PlaylistRepository
	library       music.Library
	configManager *config.Manager
}

// NewService creates a new playlists service.
func NewService(playlistRepo music.PlaylistRepository, lib music.Library, cfgManager *config.Manager) *Service {
	return &Service{
		playlistRepo:  playlistRepo,
		library:       lib,
		configManager: cfgManager,
	}
}

// CreatePlaylist creates a new playlist.
func (s *Service) CreatePlaylist(ctx context.Context, name, description string) (*music.Playlist, error) {
	slog.Debug("CreatePlaylist service called", "name", name)

	playlist := &music.Playlist{
		ID:           music.GeneratePlaylistID(),
		Name:         name,
		Description:  description,
		Tracks:       []*music.Track{},
		CreatedDate:  time.Now(),
		ModifiedDate: time.Now(),
	}

	if err := playlist.Validate(); err != nil {
		slog.Error("CreatePlaylist validation failed", "error", err)
		return nil, err
	}

	err := s.playlistRepo.Create(ctx, playlist)
	if err != nil {
		slog.Error("CreatePlaylist failed", "name", name, "error", err)
		return nil, err
	}

	slog.Debug("CreatePlaylist completed", "id", playlist.ID, "name", name)
	return playlist, nil
}

// GetPlaylist gets a playlist by ID.
func (s *Service) GetPlaylist(ctx context.Context, id string) (*music.Playlist, error) {
	slog.Debug("GetPlaylist service called", "id", id)

	playlist, err := s.playlistRepo.GetByID(ctx, id)
	if err != nil {
		slog.Error("GetPlaylist failed", "id", id, "error", err)
		return nil, err
	}

	slog.Debug("GetPlaylist completed", "id", id)
	return playlist, nil
}

// GetAllPlaylists gets all playlists.
func (s *Service) GetAllPlaylists(ctx context.Context) ([]*music.Playlist, error) {
	slog.Debug("GetAllPlaylists service called")

	playlists, err := s.playlistRepo.GetAll(ctx)
	if err != nil {
		slog.Error("GetAllPlaylists failed", "error", err)
		return nil, err
	}

	slog.Debug("GetAllPlaylists completed", "count", len(playlists))
	return playlists, nil
}

// UpdatePlaylist updates a playlist.
func (s *Service) UpdatePlaylist(ctx context.Context, playlist *music.Playlist) error {
	slog.Debug("UpdatePlaylist service called", "id", playlist.ID, "name", playlist.Name)

	playlist.ModifiedDate = time.Now()

	if err := playlist.Validate(); err != nil {
		slog.Error("UpdatePlaylist validation failed", "error", err)
		return err
	}

	err := s.playlistRepo.Update(ctx, playlist)
	if err != nil {
		slog.Error("UpdatePlaylist failed", "id", playlist.ID, "error", err)
		return err
	}

	slog.Debug("UpdatePlaylist completed", "id", playlist.ID)
	return nil
}

// DeletePlaylist deletes a playlist.
func (s *Service) DeletePlaylist(ctx context.Context, id string) error {
	slog.Debug("DeletePlaylist service called", "id", id)

	err := s.playlistRepo.Delete(ctx, id)
	if err != nil {
		slog.Error("DeletePlaylist failed", "id", id, "error", err)
		return err
	}

	slog.Debug("DeletePlaylist completed", "id", id)
	return nil
}

// AddItemToPlaylist adds tracks, artists, or albums to a playlist.
func (s *Service) AddItemToPlaylist(ctx context.Context, playlistID, itemType, itemID string) error {
	slog.Debug("AddItemToPlaylist service called", "playlistID", playlistID, "itemType", itemType, "itemID", itemID)

	// Verify playlist exists
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		slog.Error("AddItemToPlaylist: failed to get playlist", "playlistID", playlistID, "error", err)
		return fmt.Errorf("failed to get playlist %s: %w", playlistID, err)
	}
	if playlist == nil {
		slog.Error("AddItemToPlaylist: playlist not found", "playlistID", playlistID)
		return fmt.Errorf("playlist not found: %s", playlistID)
	}

	// Get track IDs to add based on item type
	var trackIDs []string
	switch itemType {
	case "track":
		// Verify track exists
		track, err := s.library.GetTrack(ctx, itemID)
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get track", "trackID", itemID, "error", err)
			return fmt.Errorf("failed to get track %s: %w", itemID, err)
		}
		if track == nil {
			slog.Error("AddItemToPlaylist: track not found in database", "trackID", itemID)
			return fmt.Errorf("track not found: %s", itemID)
		}
		trackIDs = []string{itemID}

	case "artist":
		// Get all tracks by this artist
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, 10000, 0, &music.TrackFilter{ArtistIDs: []string{itemID}})
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get artist tracks", "artistID", itemID, "error", err)
			return fmt.Errorf("failed to get artist tracks: %w", err)
		}
		for _, track := range tracks {
			trackIDs = append(trackIDs, track.ID)
		}
		if len(trackIDs) == 0 {
			return fmt.Errorf("no tracks found for artist: %s", itemID)
		}

	case "album":
		// Get all tracks from this album
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, 10000, 0, &music.TrackFilter{AlbumIDs: []string{itemID}})
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get album tracks", "albumID", itemID, "error", err)
			return fmt.Errorf("failed to get album tracks: %w", err)
		}
		for _, track := range tracks {
			trackIDs = append(trackIDs, track.ID)
		}
		if len(trackIDs) == 0 {
			return fmt.Errorf("no tracks found for album: %s", itemID)
		}

	default:
		return fmt.Errorf("unsupported item type: %s", itemType)
	}

	// Add tracks to playlist (skip duplicates)
	addedCount := 0
	for _, trackID := range trackIDs {
		err = s.playlistRepo.AddTrackToPlaylist(ctx, playlistID, trackID)
		if err != nil {
			// If track already exists, that's ok - just skip it
			if !strings.Contains(err.Error(), "already exists") {
				slog.Error("AddItemToPlaylist: failed to add track", "playlistID", playlistID, "trackID", trackID, "error", err)
				return fmt.Errorf("failed to add track %s: %w", trackID, err)
			}
		} else {
			addedCount++
		}
	}

	slog.Info("AddItemToPlaylist completed successfully", "playlistID", playlistID, "itemType", itemType, "itemID", itemID, "tracksAdded", addedCount)
	return nil
}

// RemoveTrackFromPlaylist removes a track from a playlist.
func (s *Service) RemoveTrackFromPlaylist(ctx context.Context, playlistID, trackID string) error {
	slog.Debug("RemoveTrackFromPlaylist service called", "playlistID", playlistID, "trackID", trackID)

	err := s.playlistRepo.RemoveTrackFromPlaylist(ctx, playlistID, trackID)
	if err != nil {
		slog.Error("RemoveTrackFromPlaylist failed", "playlistID", playlistID, "trackID", trackID, "error", err)
		return err
	}

	slog.Debug("RemoveTrackFromPlaylist completed", "playlistID", playlistID, "trackID", trackID)
	return nil
}

// GetPlaylistTracks gets all tracks for a playlist.
func (s *Service) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*music.Track, error) {
	slog.Debug("GetPlaylistTracks service called", "playlistID", playlistID)

	tracks, err := s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		slog.Error("GetPlaylistTracks failed", "playlistID", playlistID, "error", err)
		return nil, err
	}

	slog.Debug("GetPlaylistTracks completed", "playlistID", playlistID, "count", len(tracks))
	return tracks, nil
}

// ExportM3U exports a playlist to an M3U file.
func (s *Service) ExportM3U(ctx context.Context, playlistID, filePath string) error {
	slog.Debug("ExportM3U service called", "playlistID", playlistID, "filePath", filePath)

	// Get playlist tracks
	tracks, err := s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		slog.Error("ExportM3U: failed to get playlist tracks", "playlistID", playlistID, "error", err)
		return err
	}

	// Create output file
	file, err := os.Create(filePath)
	if err != nil {
		slog.Error("ExportM3U: failed to create file", "filePath", filePath, "error", err)
		return fmt.Errorf("failed to create M3U file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write M3U header
	_, err = writer.WriteString("#EXTM3U\n")
	if err != nil {
		return err
	}

	// Write each track
	for _, track := range tracks {
		// Write extended M3U info
		duration := track.Metadata.Duration
		artists := make([]string, len(track.Artists))
		for i, ar := range track.Artists {
			if ar.Artist != nil {
				artists[i] = ar.Artist.Name
			}
		}
		artistStr := strings.Join(artists, ", ")

		_, err = writer.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", duration, artistStr, track.Title))
		if err != nil {
			return err
		}

		// Write file path
		_, err = writer.WriteString(track.Path + "\n")
		if err != nil {
			return err
		}
	}

	err = writer.Flush()
	if err != nil {
		slog.Error("ExportM3U: failed to flush writer", "error", err)
		return err
	}

	slog.Debug("ExportM3U completed", "playlistID", playlistID, "filePath", filePath, "tracksExported", len(tracks))
	return nil
}

// GetPlaylistsContainingTrack gets all playlists that contain a specific track.
func (s *Service) GetPlaylistsContainingTrack(ctx context.Context, trackID string) ([]*music.Playlist, error) {
	slog.Debug("GetPlaylistsContainingTrack service called", "trackID", trackID)

	allPlaylists, err := s.playlistRepo.GetAll(ctx)
	if err != nil {
		slog.Error("GetPlaylistsContainingTrack: failed to get all playlists", "error", err)
		return nil, err
	}

	var containingPlaylists []*music.Playlist
	for _, playlist := range allPlaylists {
		fullPlaylist, err := s.playlistRepo.GetByID(ctx, playlist.ID)
		if err != nil {
			slog.Warn("GetPlaylistsContainingTrack: failed to get full playlist", "playlistID", playlist.ID, "error", err)
			continue
		}
		if fullPlaylist.ContainsTrack(trackID) {
			containingPlaylists = append(containingPlaylists, fullPlaylist)
		}
	}

	slog.Debug("GetPlaylistsContainingTrack completed", "trackID", trackID, "count", len(containingPlaylists))
	return containingPlaylists, nil
}
