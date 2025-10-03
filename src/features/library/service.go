package library

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/contre95/soulsolid/src/features/config"
	library "github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
)

// Service is the domain service for the library feature.
type Service struct {
	library       library.Library
	configManager *config.Manager
}

// NewService creates a new library service.
func NewService(lib library.Library, cfgManager *config.Manager) *Service {
	return &Service{
		library:       lib,
		configManager: cfgManager,
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

// GetTracks returns all tracks from the library.
func (s *Service) GetTracks(ctx context.Context) ([]*library.Track, error) {
	slog.Debug("GetTracks service called")
	tracks, err := s.library.GetTracks(ctx)
	if err != nil {
		slog.Error("GetTracks failed", "error", err)
		return nil, err
	}
	slog.Debug("GetTracks completed", "count", len(tracks))
	return tracks, nil
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

// GetArtistsPaginated returns paginated artists from the library.
func (s *Service) GetArtistsPaginated(ctx context.Context, limit, offset int) ([]*library.Artist, error) {
	slog.Debug("GetArtistsPaginated service called", "limit", limit, "offset", offset)
	artists, err := s.library.GetArtistsPaginated(ctx, limit, offset)
	if err != nil {
		slog.Error("GetArtistsPaginated failed", "error", err)
		return nil, err
	}
	slog.Debug("GetArtistsPaginated completed", "count", len(artists))
	return artists, nil
}

// GetArtistsCount returns the total count of artists in the library.
func (s *Service) GetArtistsCount(ctx context.Context) (int, error) {
	slog.Debug("GetArtistsCount service called")
	count, err := s.library.GetArtistsCount(ctx)
	if err != nil {
		slog.Error("GetArtistsCount failed", "error", err)
		return 0, err
	}
	slog.Debug("GetArtistsCount completed", "count", count)
	return count, nil
}

// GetAlbumsPaginated returns paginated albums from the library.
func (s *Service) GetAlbumsPaginated(ctx context.Context, limit, offset int) ([]*library.Album, error) {
	slog.Debug("GetAlbumsPaginated service called", "limit", limit, "offset", offset)
	albums, err := s.library.GetAlbumsPaginated(ctx, limit, offset)
	if err != nil {
		slog.Error("GetAlbumsPaginated failed", "error", err)
		return nil, err
	}
	slog.Debug("GetAlbumsPaginated completed", "count", len(albums))
	return albums, nil
}

// GetAlbumsCount returns the total count of albums in the library.
func (s *Service) GetAlbumsCount(ctx context.Context) (int, error) {
	slog.Debug("GetAlbumsCount service called")
	count, err := s.library.GetAlbumsCount(ctx)
	if err != nil {
		slog.Error("GetAlbumsCount failed", "error", err)
		return 0, err
	}
	slog.Debug("GetAlbumsCount completed", "count", count)
	return count, nil
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

// GetAlbumByArtistAndName returns an album by artist ID and album name.
func (s *Service) GetAlbumByArtistAndName(ctx context.Context, artistID, name string) (*library.Album, error) {
	slog.Debug("GetAlbumByArtistAndName service called", "artistID", artistID, "name", name)
	album, err := s.library.GetAlbumByArtistAndName(ctx, artistID, name)
	if err != nil {
		slog.Error("GetAlbumByArtistAndName failed", "artistID", artistID, "name", name, "error", err)
		return nil, err
	}
	slog.Debug("GetAlbumByArtistAndName completed", "artistID", artistID, "name", name)
	return album, nil
}

// AddAlbum adds an album to the library.
func (s *Service) AddAlbum(ctx context.Context, album *library.Album) error {
	slog.Debug("AddAlbum service called", "id", album.ID, "title", album.Title)
	err := s.library.AddAlbum(ctx, album)
	if err != nil {
		slog.Error("AddAlbum failed", "id", album.ID, "title", album.Title, "error", err)
		return err
	}
	slog.Debug("AddAlbum completed", "id", album.ID, "title", album.Title)
	return nil
}

// UpdateAlbum updates an album in the library.
func (s *Service) UpdateAlbum(ctx context.Context, album *library.Album) error {
	slog.Debug("UpdateAlbum service called", "id", album.ID, "title", album.Title)
	err := s.library.UpdateAlbum(ctx, album)
	if err != nil {
		slog.Error("UpdateAlbum failed", "id", album.ID, "title", album.Title, "error", err)
		return err
	}
	slog.Debug("UpdateAlbum completed", "id", album.ID, "title", album.Title)
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

// GetArtistByName finds an artist by name without creating it.
func (s *Service) GetArtistByName(ctx context.Context, artistName string) (*library.Artist, error) {
	artist, err := s.library.GetArtistByName(ctx, artistName)
	if err != nil {
		slog.Error("Failed to get artist by name", "artistName", artistName, "error", err)
		return nil, err
	}
	if artist == nil {
		slog.Debug("Artist not found by name", "artistName", artistName)
		return nil, nil
	}
	slog.Debug("Found artist by name", "artistName", artistName, "artistID", artist.ID)
	return artist, nil
}

// FindOrCreateArtist finds an existing artist by name or creates a new one.
func (s *Service) FindOrCreateArtist(ctx context.Context, artistName string) (*library.Artist, error) {
	slog.Debug("FindOrCreateArtist service called", "artistName", artistName)

	// First try to find existing artist
	artist, err := s.library.GetArtistByName(ctx, artistName)
	if err == nil && artist != nil {
		slog.Debug("Found existing artist", "artistName", artistName, "artistID", artist.ID)
		return artist, nil
	} else if err == nil && artist == nil {
		// Artist not found
		return nil, fmt.Errorf("artist '%s' not found in library", artistName)
	}

	// If not found, create new artist
	newArtist := &library.Artist{
		ID:   uuid.New().String(),
		Name: artistName,
	}
	err = s.library.AddArtist(ctx, newArtist)
	if err != nil {
		slog.Error("Failed to create new artist", "artistName", artistName, "error", err)
		return nil, err
	}

	slog.Debug("Created new artist", "artistName", artistName, "artistID", newArtist.ID)
	return newArtist, nil
}

// UpdateTrack updates a track in the library.
func (s *Service) UpdateTrack(ctx context.Context, track *library.Track) error {
	slog.Debug("UpdateTrack service called", "id", track.ID)
	err := s.library.UpdateTrack(ctx, track)
	if err != nil {
		slog.Error("UpdateTrack failed", "id", track.ID, "error", err)
		return err
	}
	slog.Debug("UpdateTrack completed", "id", track.ID)
	return nil
}
