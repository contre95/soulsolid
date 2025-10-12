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
	configManager  *config.Manager
	jobService     jobs.JobService
	pluginManager  *PluginManager
	tagWriter      TagWriter
	artworkService ArtworkService
}

// NewService creates a new downloading service
func NewService(cfgManager *config.Manager, jobService jobs.JobService, pluginManager *PluginManager, tagWriter TagWriter, artworkService ArtworkService) *Service {
	return &Service{
		configManager:  cfgManager,
		jobService:     jobService,
		pluginManager:  pluginManager,
		tagWriter:      tagWriter,
		artworkService: artworkService,
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

// DownloadTrack starts a download job for a track
func (s *Service) DownloadTrack(downloaderName, trackID string) (string, error) {
	_, exists := s.pluginManager.GetDownloader(downloaderName)
	if !exists {
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
