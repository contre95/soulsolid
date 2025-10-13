package downloading

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

	// Update job name with track title if it's still generic
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

	// Tag the file (artwork is already downloaded by plugin and set in track.Album.ArtworkData)
	filePath := track.Path
	slog.Debug("Tagging file", "trackID", track.ID, "filePath", filePath)
	err = e.service.tagWriter.WriteFileTags(ctx, filePath, track)
	if err != nil {
		slog.Error("Failed to tag file", "trackID", track.ID, "error", err)
		return nil, fmt.Errorf("failed to tag file: %w", err)
	}

	slog.Info("Track downloaded and tagged", "trackID", track.ID, "filePath", filePath)
	progressUpdater(100, "Track download completed")

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Warn("Failed to get file size", "filePath", filePath, "error", err)
	}
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	return map[string]any{
		"trackID":  trackID,
		"filePath": filePath,
		"fileSize": fileSize,
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
	album, err := downloader.DownloadAlbum(albumID)
	if err != nil {
		slog.Error("Failed to download album", "albumID", albumID, "error", err)
		return nil, fmt.Errorf("failed to download album: %w", err)
	}
	if album == nil {
		return nil, fmt.Errorf("album download returned nil")
	}

	// Update job name with album title if it's still generic
	if job.Name == "Download Album" {
		job.Name = fmt.Sprintf("Download: %s", album.Title)
		job.Metadata["albumTitle"] = album.Title
		slog.Info("Updated job name with album title", "jobID", job.ID, "title", album.Title)
	}
	totalTracks := len(album.Tracks)
	if totalTracks == 0 {
		progressUpdater(100, "Album download completed (no tracks)")
		return map[string]any{
			"albumID":    albumID,
			"album":      album,
			"trackCount": 0,
		}, nil
	}
	progressUpdater(20, fmt.Sprintf("Album downloaded, processing %d tracks...", totalTracks))
	slog.Debug("Creating album folder", "albumID", albumID, "artist", album.Artists[0].Artist.Name, "title", album.Title)
	// Create album folder
	albumFolderName := Sanitize(fmt.Sprintf("%s - %s", album.Artists[0].Artist.Name, album.Title))
	albumPath := filepath.Join(downloadPath, albumFolderName)
	if err := os.MkdirAll(albumPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create album directory: %w", err)
	}
	var downloadedTracks []*music.Track
	var filePaths []string
	// Download each track in the album
	for i, track := range album.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		progress := 20 + (i * 70 / totalTracks)
		progressUpdater(progress, fmt.Sprintf("Downloading track %d/%d: %s...", i+1, totalTracks, track.Title))
		slog.Debug("Downloading album track", "albumID", albumID, "trackID", track.ID, "trackNumber", i+1, "title", track.Title)
		// Download the track
		downloadedTrack, err := downloader.DownloadTrack(track.ID, albumPath, nil)
		if err != nil {
			slog.Error("Failed to download track", "trackID", track.ID, "error", err)
			continue // Continue with other tracks
		}
		if downloadedTrack == nil {
			slog.Warn("Track download returned nil", "trackID", track.ID)
			continue
		}
		slog.Debug("Processing album track metadata", "trackID", downloadedTrack.ID)
		// Enhance and validate metadata
		downloadedTrack.EnsureMetadataDefaults()
		if err := downloadedTrack.ValidateRequiredMetadata(); err != nil {
			slog.Error("Album track metadata validation failed", "trackID", downloadedTrack.ID, "error", err)
			continue // Skip this track but continue with others
		}
		slog.Debug("Preparing album track file path", "trackID", downloadedTrack.ID, "trackNumber", downloadedTrack.Metadata.TrackNumber)
		// Create filename with track number and proper extension
		trackNumber := downloadedTrack.Metadata.TrackNumber
		if trackNumber == 0 {
			trackNumber = i + 1 // Fallback to position in array
		}
		// Tag the file
		filePath := downloadedTrack.Path
		slog.Debug("Tagging album track file", "trackID", downloadedTrack.ID, "filePath", filePath)
		err = e.service.tagWriter.WriteFileTags(ctx, filePath, downloadedTrack)
		if err != nil {
			slog.Error("Failed to tag album track file", "trackID", downloadedTrack.ID, "error", err)
			continue
		}
		downloadedTracks = append(downloadedTracks, downloadedTrack)
		filePaths = append(filePaths, filePath)
		slog.Info("Track downloaded successfully", "title", downloadedTrack.Title, "filePath", filePath)
	}

	progressUpdater(100, fmt.Sprintf("Album download completed - %d tracks saved to %s", len(downloadedTracks), albumFolderName))
	return map[string]any{
		"albumID":    albumID,
		"trackCount": len(downloadedTracks),
		"filePaths":  filePaths,
		"albumPath":  albumPath,
	}, nil
}

// Cleanup performs cleanup after job completion
func (e *DownloadJobTask) Cleanup(job *jobs.Job) error {
	// TODO: Clean up temporary files, etc.
	slog.Debug("Cleaning up download job", "jobID", job.ID)
	return nil
}

// downloadImage fetches image data from URL
func downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
