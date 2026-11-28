package library

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/contre95/soulsolid/src/features/config"
	library "github.com/contre95/soulsolid/src/music"
)

// Service is the domain service for the library feature.
type Service struct {
	library       library.Library
	configManager *config.Manager
	fileManager   library.FileManager
}

// NewService creates a new library service.
func NewService(lib library.Library, cfgManager *config.Manager, fileManager library.FileManager) *Service {
	return &Service{
		library:       lib,
		configManager: cfgManager,
		fileManager:   fileManager,
	}
}

// GetDownloadsFileTree returns a tree structure of the library path.
func (s *Service) getFileTree(path string) (string, error) {
	cmd := exec.Command("tree", path)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Failed to execute tree command", "error", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("failed to run tree command: %s. Is 'tree' installed on your system?", exitErr.Stderr)
		}
		return "", err
	}
	return string(output), nil
}

// GetDownloadsFileTree returns a tree structure of the downloads path.
func (s *Service) GetDownloadsFileTree() (string, error) {
	downloadPath := s.configManager.Get().DownloadPath
	return s.getFileTree(downloadPath)
}

// GetLibraryFileTree returns a tree structure of the library path.
func (s *Service) GetLibraryFileTree() (string, error) {
	libraryPath := s.configManager.Get().LibraryPath
	return s.getFileTree(libraryPath)
}

// GetStorageSize returns the total storage size in bytes of the library.
func (s *Service) GetStorageSize(ctx context.Context) (int64, error) {
	libraryPath := s.configManager.Get().LibraryPath
	var totalSize int64

	err := filepath.Walk(libraryPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			stat, err := os.Stat(path)
			if err != nil {
				return err
			}
			totalSize += stat.Size()
		}
		return nil
	})

	if err != nil {
		slog.Error("Failed to calculate storage size", "error", err)
		return 0, err
	}

	return totalSize, nil
}

// GetArtists returns all artists from the library.
func (s *Service) GetArtists(ctx context.Context) ([]*library.Artist, error) {
	slog.Debug("GetArtists service called")
	artists, err := s.library.GetArtists(ctx)
	if err != nil {
		slog.Error("GetArtists failed", "error", err)
		return nil, err
	}
	slog.Debug("GetArtists completed", "count", len(artists))
	return artists, nil
}

// GetAlbums returns all albums from the library.
func (s *Service) GetAlbums(ctx context.Context) ([]*library.Album, error) {
	slog.Debug("GetAlbums service called")
	albums, err := s.library.GetAlbums(ctx)
	if err != nil {
		slog.Error("GetAlbums failed", "error", err)
		return nil, err
	}
	slog.Debug("GetAlbums completed", "count", len(albums))
	return albums, nil
}

// GetTracksPaginated returns paginated tracks from the library.
func (s *Service) GetTracksPaginated(ctx context.Context, limit, offset int) ([]*library.Track, error) {
	slog.Debug("GetTracksPaginated service called", "limit", limit, "offset", offset)
	tracks, err := s.library.GetTracksPaginated(ctx, limit, offset)
	if err != nil {
		slog.Error("GetTracksPaginated failed", "error", err)
		return nil, err
	}
	slog.Debug("GetTracksPaginated completed", "count", len(tracks))
	return tracks, nil
}

// GetTracksFilteredPaginated returns paginated tracks from the library with filtering.
func (s *Service) GetTracksFilteredPaginated(ctx context.Context, limit, offset int, filter *library.TrackFilter) ([]*library.Track, error) {
	slog.Debug("GetTracksFilteredPaginated service called", "limit", limit, "offset", offset, "filter", filter)
	tracks, err := s.library.GetTracksFilteredPaginated(ctx, limit, offset, filter)
	if err != nil {
		slog.Error("GetTracksFilteredPaginated failed", "error", err)
		return nil, err
	}
	slog.Debug("GetTracksFilteredPaginated completed", "count", len(tracks))
	return tracks, nil
}

// GetTracksFilteredCount returns the filtered count of tracks in the library.
func (s *Service) GetTracksFilteredCount(ctx context.Context, filter *library.TrackFilter) (int, error) {
	slog.Debug("GetTracksFilteredCount service called", "filter", filter)
	count, err := s.library.GetTracksFilteredCount(ctx, filter)
	if err != nil {
		slog.Error("GetTracksFilteredCount failed", "error", err)
		return 0, err
	}
	slog.Debug("GetTracksFilteredCount completed", "count", count)
	return count, nil
}

// GetTracksCount returns the total count of tracks in the library.
func (s *Service) GetTracksCount(ctx context.Context) (int, error) {
	slog.Debug("GetTracksCount service called")
	count, err := s.library.GetTracksCount(ctx)
	if err != nil {
		slog.Error("GetTracksCount failed", "error", err)
		return 0, err
	}
	slog.Debug("GetTracksCount completed", "count", count)
	return count, nil
}

// GetArtistsFilteredPaginated returns paginated artists from the library with filtering.
func (s *Service) GetArtistsFilteredPaginated(ctx context.Context, limit, offset int, nameFilter string) ([]*library.Artist, error) {
	slog.Debug("GetArtistsFilteredPaginated service called", "limit", limit, "offset", offset, "nameFilter", nameFilter)
	artists, err := s.library.GetArtistsFilteredPaginated(ctx, limit, offset, nameFilter)
	if err != nil {
		slog.Error("GetArtistsFilteredPaginated failed", "error", err)
		return nil, err
	}
	slog.Debug("GetArtistsFilteredPaginated completed", "count", len(artists))
	return artists, nil
}

// GetGenres returns all distinct non-empty genres in the library.
func (s *Service) GetGenres(ctx context.Context) ([]string, error) {
	return s.library.GetGenres(ctx)
}

// SearchAlbums returns albums matching query using a single lightweight JOIN query.
func (s *Service) SearchAlbums(ctx context.Context, query string, limit, offset int) ([]*library.Album, error) {
	slog.Debug("SearchAlbums service called", "query", query, "limit", limit, "offset", offset)
	albums, err := s.library.SearchAlbums(ctx, query, limit, offset)
	if err != nil {
		slog.Error("SearchAlbums failed", "error", err)
		return nil, err
	}
	slog.Debug("SearchAlbums completed", "count", len(albums))
	return albums, nil
}

// GetAlbumSummaries returns a page of albums for the album browser.
func (s *Service) GetAlbumSummaries(ctx context.Context, limit, offset int, filter *library.AlbumFilter) ([]*library.AlbumSummary, error) {
	slog.Debug("GetAlbumSummaries service called", "limit", limit, "offset", offset, "filter", filter)
	summaries, err := s.library.GetAlbumSummaries(ctx, limit, offset, filter)
	if err != nil {
		slog.Error("GetAlbumSummaries failed", "error", err)
		return nil, err
	}
	return summaries, nil
}

// GetAlbumSummariesCount returns how many albums match the filter.
func (s *Service) GetAlbumSummariesCount(ctx context.Context, filter *library.AlbumFilter) (int, error) {
	count, err := s.library.GetAlbumSummariesCount(ctx, filter)
	if err != nil {
		slog.Error("GetAlbumSummariesCount failed", "error", err)
		return 0, err
	}
	return count, nil
}

// GetAlbumTracks returns the tracks of an album in disc and track order.
func (s *Service) GetAlbumTracks(ctx context.Context, albumID string) ([]*library.Track, error) {
	tracks, err := s.library.GetAlbumTracks(ctx, albumID)
	if err != nil {
		slog.Error("GetAlbumTracks failed", "albumID", albumID, "error", err)
		return nil, err
	}
	return tracks, nil
}

// GetArtist returns a single artist from the library.
func (s *Service) GetArtist(ctx context.Context, id string) (*library.Artist, error) {
	slog.Debug("GetArtist service called", "id", id)
	artist, err := s.library.GetArtist(ctx, id)
	if err != nil {
		slog.Error("GetArtist failed", "id", id, "error", err)
		return nil, err
	}
	slog.Debug("GetArtist completed", "id", id)
	return artist, nil
}

// GetAlbum returns a single album from the library.
func (s *Service) GetAlbum(ctx context.Context, id string) (*library.Album, error) {
	slog.Debug("GetAlbum service called", "id", id)
	album, err := s.library.GetAlbum(ctx, id)
	if err != nil {
		slog.Error("GetAlbum failed", "id", id, "error", err)
		return nil, err
	}
	slog.Debug("GetAlbum completed", "id", id)
	return album, nil
}

// DeleteAlbum deletes an album from the library.
func (s *Service) DeleteAlbum(ctx context.Context, id string) error {
	slog.Debug("DeleteAlbum service called", "id", id)

	// Get all tracks in the album before deletion to access file paths
	filter := &library.TrackFilter{
		AlbumIDs: []string{id},
	}
	totalTracks, err := s.library.GetTracksFilteredCount(ctx, filter)
	if err != nil {
		slog.Error("Failed to get track count for album deletion", "albumId", id, "error", err)
		return err
	}
	albumTracks, err := s.library.GetTracksFilteredPaginated(ctx, totalTracks, 0, filter)
	if err != nil {
		slog.Error("Failed to get tracks for album deletion", "albumId", id, "error", err)
		return err
	}

	// Delete from database
	err = s.library.DeleteAlbum(ctx, id)
	if err != nil {
		slog.Error("DeleteAlbum failed", "id", id, "error", err)
		return err
	}

	// Delete track files from filesystem
	for _, track := range albumTracks {
		if err := s.fileManager.DeleteTrack(ctx, track.Path); err != nil {
			slog.Warn("Failed to delete track file from filesystem", "path", track.Path, "error", err)
			// Don't return error here - database deletion succeeded, file deletion is secondary
		} else {
			slog.Debug("Successfully deleted track file from filesystem", "path", track.Path)
		}
	}

	slog.Debug("DeleteAlbum completed", "id", id, "tracksDeleted", len(albumTracks))
	return nil
}

// GetTrack returns a single track from the library.
func (s *Service) GetTrack(ctx context.Context, id string) (*library.Track, error) {
	slog.Debug("GetTrack service called", "id", id)
	track, err := s.library.GetTrack(ctx, id)
	if err != nil {
		slog.Error("GetTrack failed", "id", id, "error", err)
		return nil, err
	}
	slog.Debug("GetTrack completed", "id", id)
	return track, nil
}

// DeleteArtist deletes an artist from the library.
func (s *Service) DeleteArtist(ctx context.Context, id string) error {
	slog.Debug("DeleteArtist service called", "id", id)

	// Get all tracks associated with this artist (direct or through albums)
	filter := &library.TrackFilter{
		ArtistIDs: []string{id},
	}
	totalTracks, err := s.library.GetTracksFilteredCount(ctx, filter)
	if err != nil {
		slog.Error("Failed to get track count for artist deletion", "artistId", id, "error", err)
		return err
	}
	artistTracks, err := s.library.GetTracksFilteredPaginated(ctx, totalTracks, 0, filter)
	if err != nil {
		slog.Error("Failed to get tracks for artist deletion", "artistId", id, "error", err)
		return err
	}

	// Delete from database (this now deletes tracks too)
	err = s.library.DeleteArtist(ctx, id)
	if err != nil {
		slog.Error("DeleteArtist failed", "id", id, "error", err)
		return err
	}

	// Delete track files from filesystem
	for _, track := range artistTracks {
		if err := s.fileManager.DeleteTrack(ctx, track.Path); err != nil {
			slog.Warn("Failed to delete track file from filesystem", "path", track.Path, "error", err)
			// Don't return error here - database deletion succeeded, file deletion is secondary
		} else {
			slog.Debug("Successfully deleted track file from filesystem", "path", track.Path)
		}
	}

	slog.Debug("DeleteArtist completed", "id", id, "tracksDeleted", len(artistTracks))
	return nil
}

// GetKeptDuplicates returns the tracks marked as intentional duplicates of the given track.
func (s *Service) GetKeptDuplicates(ctx context.Context, trackID string) ([]*library.Track, error) {
	return s.library.GetKeptDuplicates(ctx, trackID)
}

// DeleteTrack deletes a track from the library.
func (s *Service) DeleteTrack(ctx context.Context, id string) error {
	slog.Debug("DeleteTrack service called", "id", id)

	// Get track info before deletion to access the file path
	track, err := s.library.GetTrack(ctx, id)
	if err != nil {
		slog.Error("Failed to get track for deletion", "id", id, "error", err)
		return err
	}
	if track == nil {
		return fmt.Errorf("track not found: %s", id)
	}

	// Delete from database first
	err = s.library.DeleteTrack(ctx, id)
	if err != nil {
		slog.Error("DeleteTrack failed", "id", id, "error", err)
		return err
	}

	// Delete the file from filesystem
	if err := s.fileManager.DeleteTrack(ctx, track.Path); err != nil {
		slog.Warn("Failed to delete track file from filesystem", "path", track.Path, "error", err)
		// Don't return error here - database deletion succeeded, file deletion is secondary
	} else {
		slog.Debug("Successfully deleted track file from filesystem", "path", track.Path)
	}

	slog.Debug("DeleteTrack completed", "id", id)
	return nil
}
