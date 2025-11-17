package downloading

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/music"
)

// Service handles downloading operations
type Service struct {
	configManager *config.Manager
	jobService    jobs.JobService
	pluginManager *PluginManager
	tagWriter     TagWriter
}

// NewService creates a new downloading service
func NewService(cfgManager *config.Manager, jobService jobs.JobService, pluginManager *PluginManager, tagWriter TagWriter) *Service {
	return &Service{
		configManager: cfgManager,
		jobService:    jobService,
		pluginManager: pluginManager,
		tagWriter:     tagWriter,
	}
}

// SearchAlbums searches for albums
func (s *Service) SearchAlbums(downloaderName, query string, limit int) ([]music.Album, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return downloader.SearchAlbums(query, limit)
}

// SearchTracks searches for tracks
func (s *Service) SearchTracks(downloaderName, query string, limit int) ([]music.Track, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return downloader.SearchTracks(query, limit)
}

// SearchArtists searches for artists
func (s *Service) SearchArtists(downloaderName, query string, limit int) ([]music.Artist, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return downloader.SearchArtists(query, limit)
}

// SearchLinks searches for content from a direct link/URL
func (s *Service) SearchLinks(downloaderName, query string, limit int) (*LinkResult, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return downloader.SearchLinks(query, limit)
}

// DownloadTrack starts a download job for a track
func (s *Service) DownloadTrack(downloaderName, trackID string) (string, error) {
	slog.Info("DownloadTrack service", "downloaderName", downloaderName, "trackID", trackID)
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		slog.Error("Downloader not found", "downloaderName", downloaderName, "available", s.pluginManager.GetDownloaderNames())
		return "", fmt.Errorf("downloader %s not found", downloaderName)
	}

	jobID, err := s.jobService.StartJob("download_track", "Download Track", map[string]any{
		"trackID":    trackID,
		"downloader": downloaderName,
		"type":       "track",
	})
	if err != nil {
		slog.Error("Failed to start download job", "error", err)
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	return jobID, nil
}

// DownloadAlbum starts a download job for an album
func (s *Service) DownloadAlbum(downloaderName, albumID string) (string, error) {
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return "", fmt.Errorf("downloader %s not found", downloaderName)
	}

	jobID, err := s.jobService.StartJob("download_album", "Download Album", map[string]any{
		"albumID":    albumID,
		"downloader": downloaderName,
		"type":       "album",
	})
	if err != nil {
		slog.Error("Failed to start download job", "error", err)
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	return jobID, nil
}

// DownloadArtist starts a download job for an artist
func (s *Service) DownloadArtist(downloaderName, artistID string) (string, error) {
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return "", fmt.Errorf("downloader %s not found", downloaderName)
	}

	jobID, err := s.jobService.StartJob("download_artist", "Download Artist", map[string]any{
		"artistID":   artistID,
		"downloader": downloaderName,
		"type":       "artist",
	})
	if err != nil {
		slog.Error("Failed to start download job", "error", err)
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	return jobID, nil
}

// DownloadTracks starts a download job for multiple tracks
func (s *Service) DownloadTracks(downloaderName string, trackIDs []string) (string, error) {
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return "", fmt.Errorf("downloader %s not found", downloaderName)
	}

	jobID, err := s.jobService.StartJob("download_tracks", "Download Tracks", map[string]any{
		"trackIDs":   trackIDs,
		"downloader": downloaderName,
		"type":       "tracks",
	})
	if err != nil {
		slog.Error("Failed to start download job", "error", err)
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	return jobID, nil
}

// DownloadPlaylist starts a download job for a playlist
func (s *Service) DownloadPlaylist(downloaderName string, trackIDs []string, playlistName string) (string, error) {
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return "", fmt.Errorf("downloader %s not found", downloaderName)
	}

	jobID, err := s.jobService.StartJob("download_playlist", fmt.Sprintf("Download Playlist: %s", playlistName), map[string]any{
		"trackIDs":     trackIDs,
		"downloader":   downloaderName,
		"playlistName": playlistName,
		"type":         "playlist",
	})
	if err != nil {
		slog.Error("Failed to start download job", "error", err)
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	return jobID, nil
}

// GetAlbumTracks retrieves all tracks from a specific album
func (s *Service) GetAlbumTracks(downloaderName, albumID string) ([]music.Track, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}
	return downloader.GetAlbumTracks(albumID)
}

// GetChartTracks gets chart tracks from the specified downloader
func (s *Service) GetChartTracks(downloaderName string, limit int) ([]music.Track, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}
	return downloader.GetChartTracks(limit)
}

// GetDownloadPath returns the configured download path
func (s *Service) GetDownloadPath() string {
	config := s.configManager.Get()
	if config.DownloadPath == "" {
		return "./downloads"
	}
	return config.DownloadPath
}

// GetUserInfo returns the user's information for the specified downloader
func (s *Service) GetUserInfo(downloaderName string) *UserInfo {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil
	}
	return downloader.GetUserInfo()
}

// DownloaderStatus represents the status of any downloader configuration
type DownloaderStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "disabled", "invalid_credentials", "valid"
	Message string `json:"message"`
}

// GetAllDownloaders returns all loaded downloaders
func (s *Service) GetAllDownloaders() map[string]Downloader {
	return s.pluginManager.GetAllDownloaders()
}

// GetDownloaderCapabilities returns the capabilities of a specific downloader
func (s *Service) GetDownloaderCapabilities(downloaderName string) (DownloaderCapabilities, error) {
	downloader, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return DownloaderCapabilities{}, fmt.Errorf("downloader %s not found", downloaderName)
	}
	return downloader.Capabilities(), nil
}

// GetDownloaderStatuses returns the current status of all configured downloaders
func (s *Service) GetDownloaderStatuses() map[string]DownloaderStatus {
	statuses := make(map[string]DownloaderStatus)

	downloaders := s.pluginManager.GetAllDownloaders()
	for _, downloader := range downloaders {
		status := downloader.GetStatus()
		statuses[strings.ToLower(downloader.Name())] = status
	}

	return statuses
}
