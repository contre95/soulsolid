package importing

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/music"
)

var supportedExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
}

// ImportStats contains statistics about the import process
type ImportStats struct {
	Errors          int `json:"errors"`
	AlbumsImported  int `json:"albumsImported"`
	TracksImported  int `json:"tracksImported"`
	ArtistsImported int `json:"artistsImported"`
	Skipped         int `json:"skipped"`
	Queued          int `json:"queued"`
}

// Service is the domain service for the organizing feature.
type Service struct {
	fileManager       music.FileManager
	library           music.Library
	metadataReader    TagReader
	fingerprintReader FingerprintProvider
	config            *config.Manager
	jobService        jobs.JobService // TODO: Move this to domain job service
	queue             Queue
	watcher           Watcher
}

// NewService creates a new organizing service.
func NewService(lib music.Library, tagReader TagReader, fingerprintReader FingerprintProvider, fileManager music.FileManager, cfg *config.Manager, jobService jobs.JobService, queue Queue, watcher Watcher) *Service {
	s := &Service{
		config:            cfg,
		library:           lib,
		metadataReader:    tagReader,
		fingerprintReader: fingerprintReader,
		fileManager:       fileManager,
		jobService:        jobService,
		queue:             queue,
		watcher:           watcher,
	}
	if s.config.Get().Import.AutoStartWatcher {
		if err := s.StartWatcher(); err != nil {
			slog.Error("Failed to auto-start watcher", "error", err)
		}
	}
	return s
}

// ImportDirectory starts a job to import all files from a directory recursively.
func (s *Service) ImportDirectory(ctx context.Context, pathToImport string) (string, error) {
	slog.Debug("ImportDirectory service called", "path", pathToImport)
	jobID, err := s.jobService.StartJob("directory_import", "Directory Import", map[string]any{
		"path": pathToImport,
	})
	if err != nil {
		slog.Error("Service.ImportDirectory: failed to start job", "error", err)
		return "", fmt.Errorf("failed to start directory import job: %w", err)
	}
	return jobID, nil
}

// GetQueuedItems returns all items in the queue
func (s *Service) GetQueuedItems() map[string]QueueItem {
	return s.queue.GetAll()
}

// ClearQueue removes all items from the queue
func (s *Service) ClearQueue() error {
	return s.queue.Clear()
}

// PruneDownloadPath removes all supported music files from the download path and clears the queue
func (s *Service) PruneDownloadPath(ctx context.Context) error {
	downloadPath := s.config.Get().DownloadPath
	err := filepath.WalkDir(downloadPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := supportedExtensions[ext]; ok {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to prune download path: %w", err)
	}
	return s.ClearQueue()
}

// handleFileEvent handles file system events from the watcher
func (s *Service) handleFileEvent(event FileEvent) {
	slog.Info("Received file event", "path", event.Path, "type", event.EventType)
	const waitInterval = 5 * time.Second
	const maxWait = 5 * time.Minute
	start := time.Now()
	for s.jobsAreRunning() {
		if time.Since(start) > maxWait {
			slog.Info("Timed out waiting for jobs to finish, skipping watch-triggered import")
			return
		}
		slog.Info("Still waiting for jobs to finish")
		time.Sleep(waitInterval)
	}
	jobID, err := s.ImportDirectory(context.Background(), s.config.Get().DownloadPath)
	if err != nil {
		slog.Error("Failed to start watch-triggered import job", "error", err)
		return
	}
	slog.Info("Watch-triggered import job started", "jobID", jobID, "path", event.Path)
}

// StartWatcher starts the file system watcher
func (s *Service) StartWatcher() error {
	if s.watcher.IsRunning() {
		return fmt.Errorf("watcher is already running")
	}

	downloadPath := s.config.Get().DownloadPath
	if err := s.watcher.Start(context.Background(), downloadPath); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Start event handler goroutine
	go func() {
		for event := range s.watcher.GetEventChan() {
			if event.EventType == FileCreated {
				slog.Warn("File was create", "file", event.Path)
				s.handleFileEvent(event)
			} else {
				slog.Warn("Another thing happened (temp log)", "file", event.Path, "event", event.EventType)
			}
		}
	}()

	slog.Info("File watcher started")
	return nil
}

// StopWatcher stops the file system watcher
func (s *Service) StopWatcher() error {
	if !s.watcher.IsRunning() {
		return fmt.Errorf("watcher is not running")
	}
	if s.watcher != nil {
		s.watcher.Stop()
	}
	slog.Info("File watcher stopped")
	return nil
}

// GetWatcherStatus returns the current status of the watcher
func (s *Service) GetWatcherStatus() bool {
	return s.watcher.IsRunning()
}

// jobsAreRunning checks if there are any running jobs
func (s *Service) jobsAreRunning() bool {
	jobs := s.jobService.GetJobs()
	for _, job := range jobs {
		if job.Status == "running" {
			return true
		}
	}
	return false
}

// ProcessQueueItem processes a single queue item
func (s *Service) ProcessQueueItem(ctx context.Context, itemID string, action string) error {
	item, err := s.queue.GetByID(itemID)
	if err != nil || item.Track == nil {
		return fmt.Errorf("queue item not found: %w", err)
	}
	switch action {
	case "cancel":
		return s.queue.Remove(itemID)
	case "replace":
		// For replace action, we need to find the existing track to replace
		// Use fingerprint to find the existing track
		existingTrack, err := s.library.GetTrack(ctx, item.Track.ID)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return fmt.Errorf("no existing track found with matching ID for replacement")
			}
			return fmt.Errorf("failed to find existing track for replacement: %w", err)
		} else if existingTrack == nil {
			return fmt.Errorf("no existing track found with matching ID for replacement")
		}
		move := s.config.Get().Import.Move
		if err := s.replaceTrack(ctx, item.Track, existingTrack, move, nil); err != nil {
			return fmt.Errorf("failed to replace track: %w", err)
		}
		return s.queue.Remove(itemID)
	case "import":
		moveFiles := s.config.Get().Import.Move
		if err := s.importTrack(ctx, item.Track, moveFiles, nil); err != nil {
			return fmt.Errorf("failed to import track: %w", err)
		}
		return s.queue.Remove(itemID)
	case "delete":
		// Delete the file from the import location
		if err := s.fileManager.DeleteTrack(ctx, item.Track.Path); err != nil {
			return fmt.Errorf("failed to delete track file: %w", err)
		}
		return s.queue.Remove(itemID)
	default:
		return fmt.Errorf("Invalid action %s. Should be one of %s", action, "import,replace,cancel,delete")
	}

}

// GetGroupedByArtist returns queue items grouped by artist
func (s *Service) GetGroupedByArtist() map[string][]QueueItem {
	return s.queue.GetGroupedByArtist()
}

// GetGroupedByAlbum returns queue items grouped by album
func (s *Service) GetGroupedByAlbum() map[string][]QueueItem {
	return s.queue.GetGroupedByAlbum()
}

// ProcessQueueGroup processes all items in a group with the given action
func (s *Service) ProcessQueueGroup(ctx context.Context, groupKey string, groupType string, action string) error {
	// Get the group items first to process them individually
	var groupItems []QueueItem

	if groupType == "artist" {
		groups := s.GetGroupedByArtist()
		groupItems = groups[groupKey]
	} else if groupType == "album" {
		groups := s.GetGroupedByAlbum()
		groupItems = groups[groupKey]
	} else {
		return fmt.Errorf("invalid group type: %s", groupType)
	}

	if len(groupItems) == 0 {
		return fmt.Errorf("no items found in group %s", groupKey)
	}

	// Process each item in the group
	for _, item := range groupItems {
		if err := s.ProcessQueueItem(ctx, item.ID, action); err != nil {
			slog.Error("Failed to process queue item in group", "itemID", item.ID, "action", action, "error", err)
			// Continue processing other items even if one fails
		}
	}

	return nil
}

// replaceTrack handles replacing an existing track with a new one
func (s *Service) replaceTrack(ctx context.Context, newTrack, existingTrack *music.Track, move bool, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	// First organize the new file to library location
	var newPath string
	var err error
	if move {
		newPath, err = s.fileManager.MoveTrack(ctx, newTrack)
	} else {
		newPath, err = s.fileManager.CopyTrack(ctx, newTrack)
	}
	if err != nil {
		return fmt.Errorf("could not organize replacement track: %w", err)
	}
	oldPath := existingTrack.Path
	existingTrack.Path = newPath
	existingTrack.Metadata = newTrack.Metadata
	existingTrack.Title = newTrack.Title
	existingTrack.TitleVersion = newTrack.TitleVersion
	if err := s.populateTrackArtistsAndAlbum(ctx, newTrack, logger); err != nil {
		return err
	}
	existingTrack.Artists = newTrack.Artists
	existingTrack.Album = newTrack.Album
	if err := existingTrack.Validate(); err != nil {
		logger.Error("Service.replaceTrack: existing track validation failed after update", "error", err, "title", existingTrack.Title)
		return fmt.Errorf("existing track validation failed: %w", err)
	}
	// Update the track in the database
	if err := s.library.UpdateTrack(ctx, existingTrack); err != nil {
		return fmt.Errorf("failed to update existing track for replacement: %w", err)
	}
	// Delete the old file if paths differ (to avoid lingering files)
	if oldPath != newPath {
		err := s.fileManager.DeleteTrack(ctx, oldPath)
		if err != nil {
			logger.Warn("failed deleting old path of replaced track", "track", newTrack, "newPath", newPath, "oldPath", oldPath)
		}
	}
	return nil
}

// populateTrackArtistsAndAlbum populates the Artists and Album fields of a track with database references
func (s *Service) populateTrackArtistsAndAlbum(ctx context.Context, track *music.Track, logger *slog.Logger) error {
	// Create/find artist if it doesn't exist
	if len(track.Artists) > 0 {
		for i, artistRole := range track.Artists {
			artist, err := s.library.FindOrCreateArtist(ctx, artistRole.Artist.Name)
			if err != nil {
				logger.Error("Service.populateTrackArtistsAndAlbum: failed to find/create artist", "error", err, "artist", artistRole.Artist.Name, "title", track.Title)
				return fmt.Errorf("failed to find/create artist %s: %w", artistRole.Artist.Name, err)
			}
			track.Artists[i].Artist = artist
		}
	}

	// Create/find album if it doesn't exist
	if track.Album != nil && track.Album.Title != "" {
		var artist *music.Artist
		// Use album artists if available, otherwise fall back to first track artist
		if len(track.Album.Artists) > 0 {
			// Ensure album artists exist in database
			for i, artistRole := range track.Album.Artists {
				dbArtist, err := s.library.FindOrCreateArtist(ctx, artistRole.Artist.Name)
				if err != nil {
					logger.Error("Service.populateTrackArtistsAndAlbum: failed to find/create album artist", "error", err, "artist", artistRole.Artist.Name, "album", track.Album.Title, "title", track.Title)
					return fmt.Errorf("failed to find/create album artist %s: %w", artistRole.Artist.Name, err)
				}
				track.Album.Artists[i].Artist = dbArtist
			}
			artist = track.Album.Artists[0].Artist
		} else if len(track.Artists) > 0 {
			artist = track.Artists[0].Artist
		} else {
			logger.Error("Service.populateTrackArtistsAndAlbum: no artists available for album", "album", track.Album.Title, "title", track.Title)
			return fmt.Errorf("no artists available for album %s", track.Album.Title)
		}
		album, err := s.library.FindOrCreateAlbum(ctx, artist, track.Album.Title, track.Metadata.Year)
		if err != nil {
			logger.Error("Service.populateTrackArtistsAndAlbum: failed to find/create album", "error", err, "album", track.Album.Title, "title", track.Title)
			return fmt.Errorf("failed to find/create album %s: %w", track.Album.Title, err)
		}
		track.Album = album
	}

	return nil
}

// importTrack handles the import process for a track (generic method used by both directory import and queue processing)
func (s *Service) importTrack(ctx context.Context, track *music.Track, move bool, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	var newPath string
	var err error

	if move {
		newPath, err = s.fileManager.MoveTrack(ctx, track)
	} else {
		newPath, err = s.fileManager.CopyTrack(ctx, track)
	}
	if err != nil {
		logger.Error("Service.importTrack: could not organize track", "error", err, "title", track.Title)
		return fmt.Errorf("could not organize track: %w", err)
	}
	track.Path = newPath

	if err := s.populateTrackArtistsAndAlbum(ctx, track, logger); err != nil {
		return err
	}

	if err := track.Validate(); err != nil {
		logger.Error("Service.importTrack: track validation failed after population", "error", err, "title", track.Title)
		return fmt.Errorf("track validation failed: %w", err)
	}

	// Check if track already exists
	existingTrack, err := s.library.FindTrackByPath(ctx, track.Path)
	if err != nil && err.Error() != "sql: no rows in result set" {
		logger.Error("Service.importTrack: failed to check if track exists", "error", err, "path", track.Path)
		return fmt.Errorf("failed to check if track exists: %w", err)
	}

	// If track already exists, treat as successfully imported
	if existingTrack != nil {
		logger.Info("Track already exists, skipping import", "path", track.Path, "title", track.Title)
		return nil
	}

	// Add track to database
	if err := s.library.AddTrack(ctx, track); err != nil {
		logger.Error("Service.importTrack: failed to add track to database", "error", err, "title", track.Title)
		return fmt.Errorf("failed to add track to database: %w", err)
	}

	return nil
}
