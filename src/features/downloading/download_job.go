package downloading

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/music"
)

// Sanitize creates a filesystem-safe filename
func Sanitize(filename string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := re.ReplaceAllString(filename, " ")
	sanitized = strings.Trim(sanitized, " .")
	sanitized = regexp.MustCompile(`\s+`).ReplaceAllString(sanitized, " ")
	return sanitized
}

// DownloadJobTask handles download job execution
type DownloadJobTask struct {
	service *Service
}

// NewDownloadJobTask creates a new download job Task
func NewDownloadJobTask(service *Service) *DownloadJobTask {
	return &DownloadJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for download jobs
func (e *DownloadJobTask) MetadataKeys() []string {
	return []string{"type"}
}

// Execute performs the download operation
func (e *DownloadJobTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	jobType, ok := job.Metadata["type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid job type")
	}
	downloadPath := e.service.configManager.Get().DownloadPath
	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}
	switch jobType {
	case "track":
		return e.executeTrackDownload(ctx, job, progressUpdater, downloadPath)
	case "album":
		return e.executeAlbumDownload(ctx, job, progressUpdater, downloadPath)
	case "artist":
		return e.executeArtistDownload(ctx, job, progressUpdater, downloadPath)
	case "tracks":
		return e.executeTracksDownload(ctx, job, progressUpdater, downloadPath)
	default:
		return nil, fmt.Errorf("unsupported download type: %s", jobType)
	}
}

// executeTrackDownload handles track download jobs
func (e *DownloadJobTask) executeTrackDownload(ctx context.Context, job *jobs.Job, progressUpdater func(int, string), downloadPath string) (map[string]any, error) {
	trackID, ok := job.Metadata["trackID"].(string)
	if !ok {
		return nil, fmt.Errorf("trackID not found in job metadata")
	}

	downloaderName, ok := job.Metadata["downloader"].(string)
	if !ok {
		return nil, fmt.Errorf("downloader not found in job metadata")
	}

	downloader, exists := e.service.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	slog.Debug("Starting track download job", "trackID", trackID, "downloader", downloaderName, "jobID", job.ID)
	progressUpdater(10, fmt.Sprintf("Starting %s track download...", trackID))

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	progressUpdater(25, fmt.Sprintf("Downloading track from %s", downloader.Name()))

	// Create progress callback for download phase (25%-75% of total progress)
	downloadProgressCallback := func(downloaded, total int64) {
		if total > 0 {
			// Map download progress to 25%-75% range
			downloadProgress := float64(downloaded) / float64(total)
			overallProgress := 25 + int(downloadProgress*50) // 25% + (0-50% based on download)
			progressUpdater(overallProgress, fmt.Sprintf("Downloading... %.1f%% (%.1f MB / %.1f MB)",
				downloadProgress*100,
				float64(downloaded)/(1024*1024),
				float64(total)/(1024*1024)))
		}
	}

	track, err := downloader.DownloadTrack(trackID, downloadPath, downloadProgressCallback)
	if err != nil {
		slog.Error("Failed to download track", "trackID", trackID, "error", err)
		return nil, fmt.Errorf("failed to download track: %w", err)
	}
	// Print track pretty for debugging
	slog.Debug("Track downloaded", "track", track.Pretty())

	job.Name = fmt.Sprintf("Download: %s (with %s)", track.Title, track.Artists[0].Artist.Name)
	job.Metadata["trackTitle"] = track.Title
	slog.Info("Updated job name with track title", "jobID", job.ID, "title", track.Title)

	progressUpdater(75, "Track downloaded, processing metadata...")

	// Enhance track metadata with fallbacks
	slog.Debug("Enhancing track metadata with fallbacks", "trackID", track.ID)
	track.EnsureMetadataDefaults()

	// Validate required metadata
	slog.Debug("Validating required metadata", "trackID", track.ID)
	if err := track.ValidateRequiredMetadata(); err != nil {
		slog.Error("Metadata validation failed", "trackID", track.ID, "error", err)
		return nil, fmt.Errorf("metadata validation failed: %w", err)
	}

	filePath := track.Path
	cfg := e.service.configManager.Get()
	if cfg.Downloaders.TagFile {
		// Tag the file
		slog.Debug("Tagging file", "trackID", track.ID, "filePath", filePath)
		err = e.service.tagWriter.WriteFileTags(ctx, filePath, track)
		if err != nil {
			slog.Error("Failed to tag file", "trackID", track.ID, "error", err)
			return nil, fmt.Errorf("failed to tag file: %w", err)
		}

		slog.Info("Track downloaded, tagged and artwork embedded", "trackID", track.ID, "filePath", filePath)
	} else {
		slog.Info("Track downloaded without tagging or artwork embedding", "trackID", track.ID, "filePath", filePath)
	}
	progressUpdater(100, "Track download completed")

	return map[string]any{
		"trackID":  trackID,
		"filePath": filePath,
	}, nil
}

// executeAlbumDownload handles album download jobs
func (e *DownloadJobTask) executeAlbumDownload(ctx context.Context, job *jobs.Job, progressUpdater func(int, string), downloadPath string) (map[string]any, error) {
	albumID, ok := job.Metadata["albumID"].(string)
	if !ok {
		return nil, fmt.Errorf("albumID not found in job metadata")
	}

	downloaderName, ok := job.Metadata["downloader"].(string)
	if !ok {
		return nil, fmt.Errorf("downloader not found in job metadata")
	}

	downloader, exists := e.service.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	slog.Debug("Starting album download job", "albumID", albumID, "downloader", downloaderName, "jobID", job.ID)
	progressUpdater(5, "Starting album download...")
	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	progressUpdater(10, fmt.Sprintf("Downloading album from %s...", downloader.Name()))
	tracks, err := downloader.DownloadAlbum(albumID, downloadPath, func(downloaded, total int64) {
		// Convert plugin progress (0-100) to job progress (10-90)
		jobProgress := 10 + (downloaded * 80 / total)
		progressUpdater(int(jobProgress), fmt.Sprintf("Downloading album from %s... (%d%%)", downloader.Name(), downloaded*100/total))
	})
	if err != nil {
		slog.Error("Failed to download album", "albumID", albumID, "error", err)
		return nil, fmt.Errorf("failed to download album: %w", err)
	}
	if len(tracks) == 0 {
		progressUpdater(100, "Album download completed (no tracks)")
		return map[string]any{
			"albumID":    albumID,
			"trackCount": 0,
		}, nil
	}

	// Update job name with album title if it's still generic (extract from first track)
	if job.Name == "Download Album" && len(tracks) > 0 && tracks[0].Album != nil {
		albumTitle := tracks[0].Album.Title
		job.Name = fmt.Sprintf("Download: %s", albumTitle)
		job.Metadata["albumTitle"] = albumTitle
		slog.Info("Updated job name with album title", "jobID", job.ID, "title", albumTitle)
	}

	totalTracks := len(tracks)
	progressUpdater(20, fmt.Sprintf("Album downloaded, processing %d tracks...", totalTracks))

	var downloadedTracks []*music.Track
	var filePaths []string
	// Process each downloaded track
	for i, track := range tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		progress := 20 + (i * 70 / totalTracks)
		progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s...", i+1, totalTracks, track.Title))
		slog.Debug("Processing album track", "albumID", albumID, "trackID", track.ID, "trackNumber", i+1, "title", track.Title)

		// Track is already downloaded by plugin, just validate and tag
		slog.Debug("Processing album track metadata", "trackID", track.ID, "hasAlbum", track.Album != nil, "hasArtwork", track.Album != nil && len(track.Album.ArtworkData) > 0)
		// Enhance and validate metadata
		track.EnsureMetadataDefaults()
		if err := track.ValidateRequiredMetadata(); err != nil {
			slog.Error("Album track metadata validation failed", "trackID", track.ID, "error", err)
			continue // Skip this track but continue with others
		}

		cfg := e.service.configManager.Get()
		if cfg.Downloaders.TagFile {
			// Tag the file (artwork is already downloaded by plugin and set in track.Album.ArtworkData)
			filePath := track.Path
			slog.Debug("Tagging album track file", "trackID", track.ID, "filePath", filePath, "title", track.Title, "artist", track.Artists[0].Artist.Name)

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				slog.Error("Track file does not exist for tagging", "trackID", track.ID, "filePath", filePath)
				continue
			}

			err = e.service.tagWriter.WriteFileTags(ctx, filePath, track)
			if err != nil {
				slog.Error("Failed to tag album track file", "trackID", track.ID, "filePath", filePath, "error", err)
				continue
			}

			slog.Info("Track processed successfully", "title", track.Title, "filePath", filePath)
		} else {
			slog.Info("Track downloaded without tagging or artwork embedding", "trackID", track.ID, "filePath", track.Path)
		}
		downloadedTracks = append(downloadedTracks, track)
		filePaths = append(filePaths, track.Path)
	}

	// Extract album path from the first track's directory
	albumPath := ""
	if len(downloadedTracks) > 0 {
		albumPath = filepath.Dir(downloadedTracks[0].Path)
	}

	progressUpdater(100, fmt.Sprintf("Album download completed - %d tracks processed", len(downloadedTracks)))
	return map[string]any{
		"albumID":    albumID,
		"trackCount": len(downloadedTracks),
		"filePaths":  filePaths,
		"albumPath":  albumPath,
	}, nil
}

// executeArtistDownload handles artist download jobs
func (e *DownloadJobTask) executeArtistDownload(ctx context.Context, job *jobs.Job, progressUpdater func(int, string), downloadPath string) (map[string]any, error) {
	artistID, ok := job.Metadata["artistID"].(string)
	if !ok {
		return nil, fmt.Errorf("artistID not found in job metadata")
	}

	downloaderName, ok := job.Metadata["downloader"].(string)
	if !ok {
		return nil, fmt.Errorf("downloader not found in job metadata")
	}

	downloader, exists := e.service.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	slog.Debug("Starting artist download job", "artistID", artistID, "downloader", downloaderName, "jobID", job.ID)
	progressUpdater(5, "Starting artist download...")

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	progressUpdater(10, fmt.Sprintf("Downloading artist from %s...", downloader.Name()))
	tracks, err := downloader.DownloadArtist(artistID, downloadPath, func(downloaded, total int64) {
		// Convert plugin progress (0-100) to job progress (10-90)
		jobProgress := 10 + (downloaded * 80 / total)
		progressUpdater(int(jobProgress), fmt.Sprintf("Downloading artist from %s... (%d%%)", downloader.Name(), downloaded*100/total))
	})
	if err != nil {
		slog.Error("Failed to download artist", "artistID", artistID, "error", err)
		return nil, fmt.Errorf("failed to download artist: %w", err)
	}

	if len(tracks) == 0 {
		progressUpdater(100, "Artist download completed (no tracks)")
		return map[string]any{
			"artistID":   artistID,
			"trackCount": 0,
		}, nil
	}

	// Update job name with artist name if available (extract from first track)
	if job.Name == "Download Artist" && len(tracks) > 0 && len(tracks[0].Artists) > 0 {
		artistName := tracks[0].Artists[0].Artist.Name
		job.Name = fmt.Sprintf("Download: %s (Artist)", artistName)
		job.Metadata["artistName"] = artistName
		slog.Info("Updated job name with artist name", "jobID", job.ID, "name", artistName)
	}

	totalTracks := len(tracks)
	progressUpdater(20, fmt.Sprintf("Artist downloaded, processing %d tracks...", totalTracks))

	var downloadedTracks []*music.Track
	var filePaths []string

	// Process each downloaded track
	for i, track := range tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		progress := 20 + (i * 70 / totalTracks)
		progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s...", i+1, totalTracks, track.Title))
		slog.Debug("Processing artist track", "artistID", artistID, "trackID", track.ID, "trackNumber", i+1, "title", track.Title)

		// Track is already downloaded by plugin, just validate and tag
		slog.Debug("Processing artist track metadata", "trackID", track.ID, "hasAlbum", track.Album != nil, "hasArtwork", track.Album != nil && len(track.Album.ArtworkData) > 0)

		// Enhance and validate metadata
		track.EnsureMetadataDefaults()
		if err := track.ValidateRequiredMetadata(); err != nil {
			slog.Error("Artist track metadata validation failed", "trackID", track.ID, "error", err)
			continue // Skip this track but continue with others
		}

		cfg := e.service.configManager.Get()
		if cfg.Downloaders.TagFile {
			// Tag the file
			filePath := track.Path
			slog.Debug("Tagging artist track file", "trackID", track.ID, "filePath", filePath, "title", track.Title, "artist", track.Artists[0].Artist.Name)

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				slog.Error("Track file does not exist for tagging", "trackID", track.ID, "filePath", filePath)
				continue
			}

			err = e.service.tagWriter.WriteFileTags(ctx, filePath, track)
			if err != nil {
				slog.Error("Failed to tag artist track file", "trackID", track.ID, "filePath", filePath, "error", err)
				continue
			}

			slog.Info("Track processed successfully", "title", track.Title, "filePath", filePath)
		} else {
			slog.Info("Track downloaded without tagging or artwork embedding", "trackID", track.ID, "filePath", track.Path)
		}

		downloadedTracks = append(downloadedTracks, track)
		filePaths = append(filePaths, track.Path)
	}

	// Extract artist path from the first track's parent directory
	artistPath := ""
	if len(downloadedTracks) > 0 {
		// Artist path is two levels up from the track (track is in Artist/Album/track.ext)
		artistPath = filepath.Dir(filepath.Dir(downloadedTracks[0].Path))
	}

	progressUpdater(100, fmt.Sprintf("Artist download completed - %d tracks processed", len(downloadedTracks)))
	return map[string]any{
		"artistID":   artistID,
		"trackCount": len(downloadedTracks),
		"filePaths":  filePaths,
		"artistPath": artistPath,
	}, nil
}

// executeTracksDownload handles multiple track download jobs
func (e *DownloadJobTask) executeTracksDownload(ctx context.Context, job *jobs.Job, progressUpdater func(int, string), downloadPath string) (map[string]any, error) {
	var trackIDs []string

	// Try to get trackIDs as []string first
	if ids, ok := job.Metadata["trackIDs"].([]string); ok {
		trackIDs = ids
	} else if idsInterface, ok := job.Metadata["trackIDs"].([]interface{}); ok {
		// Handle as []interface{}
		for _, id := range idsInterface {
			if idStr, ok := id.(string); ok {
				trackIDs = append(trackIDs, idStr)
			}
		}
	} else if trackIDsStr, ok := job.Metadata["trackIDs"].(string); ok {
		// Fallback: if stored as comma-separated string
		trackIDs = strings.Split(trackIDsStr, ",")
		for i, id := range trackIDs {
			trackIDs[i] = strings.TrimSpace(id)
		}
	} else {
		return nil, fmt.Errorf("trackIDs not found in job metadata or invalid format")
	}

	if len(trackIDs) == 0 {
		return nil, fmt.Errorf("no track IDs provided")
	}

	downloaderName, ok := job.Metadata["downloader"].(string)
	if !ok {
		return nil, fmt.Errorf("downloader not found in job metadata")
	}

	downloader, exists := e.service.pluginManager.GetDownloader(downloaderName)
	if !exists {
		return nil, fmt.Errorf("downloader %s not found", downloaderName)
	}

	slog.Debug("Starting tracks download job", "trackIDs", trackIDs, "downloader", downloaderName, "jobID", job.ID)
	progressUpdater(5, fmt.Sprintf("Starting download of %d tracks...", len(trackIDs)))

	totalTracks := len(trackIDs)
	var downloadedTracks []*music.Track
	var filePaths []string

	// Process each track
	for i, trackID := range trackIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		progress := 5 + (i * 90 / totalTracks)
		progressUpdater(progress, fmt.Sprintf("Downloading track %d/%d: %s...", i+1, totalTracks, trackID))

		slog.Debug("Downloading track", "trackID", trackID, "downloader", downloaderName, "jobID", job.ID)

		// Download the track
		track, err := downloader.DownloadTrack(trackID, downloadPath, func(downloaded, total int64) {
			// Track-level progress (small portion of overall progress)
			trackProgress := progress + int(downloaded*5/total)
			progressUpdater(trackProgress, fmt.Sprintf("Downloading track %d/%d... (%d%%)", i+1, totalTracks, downloaded*100/total))
		})
		if err != nil {
			slog.Error("Failed to download track", "trackID", trackID, "error", err)
			continue // Skip this track but continue with others
		}

		// Process the downloaded track
		track.EnsureMetadataDefaults()
		if err := track.ValidateRequiredMetadata(); err != nil {
			slog.Error("Track metadata validation failed", "trackID", trackID, "error", err)
			continue
		}

		cfg := e.service.configManager.Get()
		if cfg.Downloaders.TagFile {
			filePath := track.Path
			slog.Debug("Tagging track file", "trackID", trackID, "filePath", filePath, "title", track.Title)

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				slog.Error("Track file does not exist for tagging", "trackID", trackID, "filePath", filePath)
				continue
			}

			err = e.service.tagWriter.WriteFileTags(ctx, filePath, track)
			if err != nil {
				slog.Error("Failed to tag track file", "trackID", trackID, "filePath", filePath, "error", err)
				continue
			}

			slog.Info("Track processed successfully", "title", track.Title, "filePath", filePath)
		}

		downloadedTracks = append(downloadedTracks, track)
		filePaths = append(filePaths, track.Path)
	}

	progressUpdater(100, fmt.Sprintf("Tracks download completed - %d tracks processed", len(downloadedTracks)))
	return map[string]any{
		"trackIDs":   trackIDs,
		"trackCount": len(downloadedTracks),
		"filePaths":  filePaths,
	}, nil
}

// Cleanup performs cleanup after job completion
func (e *DownloadJobTask) Cleanup(job *jobs.Job) error {
	// TODO: Clean up temporary files, etc.
	slog.Debug("Cleaning up download job", "jobID", job.ID)
	return nil
}
