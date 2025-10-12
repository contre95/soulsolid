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

// validateRequiredMetadata ensures all required metadata fields are present
func validateRequiredMetadata(track *music.Track) error {
	var missingFields []string
	if len(track.Artists) == 0 || track.Artists[0].Artist.Name == "" {
		missingFields = append(missingFields, "Artist")
	}
	if track.Album == nil || track.Album.Title == "" {
		missingFields = append(missingFields, "Album")
	}
	if track.Metadata.Year == 0 {
		missingFields = append(missingFields, "Year")
	}
	if len(missingFields) > 0 {
		return fmt.Errorf("missing required metadata fields: %s", strings.Join(missingFields, ", "))
	}
	return nil
}

// replaceNonExistentMetadata adds fallback values for missing metadata
func replaceNonExistentMetadata(track *music.Track) {
	// Fallback for missing artist
	if len(track.Artists) == 0 || track.Artists[0].Artist.Name == "" {
		slog.Warn("Missing artist metadata, using fallback", "trackID", track.ID)
		track.Artists = []music.ArtistRole{{
			Artist: &music.Artist{Name: "Unknown Artist"},
			Role:   "main",
		}}
	}
	// Fallback for missing album
	if track.Album == nil || track.Album.Title == "" {
		slog.Warn("Missing album metadata, using fallback", "trackID", track.ID)
		track.Album = &music.Album{Title: "Unknown Album"}
	}
	// Fallback for missing year
	if track.Metadata.Year == 0 {
		slog.Warn("Missing year metadata, using current year", "trackID", track.ID)
		track.Metadata.Year = 0000
	}
	// Fallback for missing genre
	if track.Metadata.Genre == "" {
		slog.Warn("Missing genre metadata, using fallback", "trackID", track.ID)
		track.Metadata.Genre = "Unknown"
	}
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

	track, err := downloader.DownloadTrack(trackID, downloadProgressCallback)
	if err != nil {
		slog.Error("Failed to download track", "trackID", trackID, "error", err)
		return nil, fmt.Errorf("failed to download track: %w", err)
	}

	// Update job name with track title if it's still generic
	if job.Name == "Download Track" {
		job.Name = fmt.Sprintf("Download: %s (with %s)", track.Title, track.Artists[0].Artist.Name)
		job.Metadata["trackTitle"] = track.Title
		slog.Info("Updated job name with track title", "jobID", job.ID, "title", track.Title)
	}

	progressUpdater(75, "Track downloaded, embedding metadata...")

	// Enhance track metadata with fallbacks
	slog.Debug("Enhancing track metadata with fallbacks", "trackID", track.ID)
	replaceNonExistentMetadata(track)

	// Validate required metadata
	slog.Debug("Validating required metadata", "trackID", track.ID)
	if err := validateRequiredMetadata(track); err != nil {
		slog.Error("Metadata validation failed", "trackID", track.ID, "error", err)
		return nil, fmt.Errorf("metadata validation failed: %w", err)
	}

	// Sanitize filename with proper extension based on format
	slog.Debug("Preparing file path", "trackID", track.ID, "format", track.Format)
	extension := "." + track.Format
	safeFileName := Sanitize(track.Title) + extension
	filePath := filepath.Join(downloadPath, safeFileName)

	// Download artwork for embedding if enabled
	var artworkCleanup func()
	if e.service.configManager.Get().Downloaders.Artwork.Embedded.Enabled {
		artworkPath, cleanup, err := e.service.artworkService.GetArtworkForTrack(ctx, track)
		if err != nil {
			slog.Warn("Failed to download artwork for embedding", "trackID", track.ID, "error", err)
		} else if artworkPath != "" {
			// Set artwork path for tag writer to use
			if track.Album == nil {
				track.Album = &music.Album{}
			}
			track.Album.ArtworkPath = artworkPath
			artworkCleanup = cleanup
		}
	}

	// Tag the audio data in memory
	slog.Debug("Tagging audio data in memory", "trackID", track.ID, "fileSize", len(track.Data))
	taggedData, err := e.service.tagWriter.TagAudioData(ctx, track.Data, track)
	if err != nil {
		slog.Error("Failed to tag audio data", "trackID", track.ID, "error", err)
		if artworkCleanup != nil {
			artworkCleanup()
		}
		return nil, fmt.Errorf("failed to tag audio data: %w", err)
	}

	// Write tagged data to file
	slog.Debug("Writing tagged data to file", "trackID", track.ID, "filePath", filePath, "originalSize", len(track.Data), "taggedSize", len(taggedData))
	err = os.WriteFile(filePath, taggedData, 0644)
	if err != nil {
		slog.Error("Failed to save tagged track to disk", "filePath", filePath, "error", err)
		if artworkCleanup != nil {
			artworkCleanup()
		}
		return nil, fmt.Errorf("failed to save tagged track: %w", err)
	}

	// Clean up temp artwork file
	if artworkCleanup != nil {
		artworkCleanup()
	}

	// Save local artwork file if enabled
	if err := e.service.artworkService.SaveLocalArtwork(ctx, track, filepath.Dir(filePath)); err != nil {
		slog.Warn("Failed to save local artwork", "trackID", track.ID, "error", err)
		// Don't fail the download for artwork issues
	}

	slog.Info("Track downloaded and tagged", "trackID", track.ID, "filePath", filePath)
	progressUpdater(100, "Track download completed")

	return map[string]any{
		"trackID":  trackID,
		"filePath": filePath,
		"fileSize": len(track.Data),
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
		downloadedTrack, err := downloader.DownloadTrack(track.ID, nil)
		if err != nil {
			slog.Error("Failed to download track", "trackID", track.ID, "error", err)
			continue // Continue with other tracks
		}
		if downloadedTrack == nil || len(downloadedTrack.Data) == 0 {
			slog.Warn("Track download returned empty data", "trackID", track.ID)
			continue
		}
		slog.Debug("Processing album track metadata", "trackID", downloadedTrack.ID)
		// Enhance and validate metadata
		replaceNonExistentMetadata(downloadedTrack)
		if err := validateRequiredMetadata(downloadedTrack); err != nil {
			slog.Error("Album track metadata validation failed", "trackID", downloadedTrack.ID, "error", err)
			continue // Skip this track but continue with others
		}
		slog.Debug("Preparing album track file path", "trackID", downloadedTrack.ID, "trackNumber", downloadedTrack.Metadata.TrackNumber)
		// Create filename with track number and proper extension
		trackNumber := downloadedTrack.Metadata.TrackNumber
		if trackNumber == 0 {
			trackNumber = i + 1 // Fallback to position in array
		}
		extension := "." + downloadedTrack.Format
		safeFileName := fmt.Sprintf("%02d - %s%s", trackNumber, Sanitize(downloadedTrack.Title), extension)
		filePath := filepath.Join(albumPath, safeFileName)
		// Tag the audio data in memory
		slog.Debug("Tagging album track data in memory", "trackID", downloadedTrack.ID, "fileSize", len(downloadedTrack.Data))
		taggedData, err := e.service.tagWriter.TagAudioData(ctx, downloadedTrack.Data, downloadedTrack)
		if err != nil {
			slog.Error("Failed to tag album track data", "trackID", downloadedTrack.ID, "error", err)
			continue
		}

		// Write tagged data to file
		slog.Debug("Writing tagged album track to file", "trackID", downloadedTrack.ID, "filePath", filePath, "originalSize", len(downloadedTrack.Data), "taggedSize", len(taggedData))
		err = os.WriteFile(filePath, taggedData, 0644)
		if err != nil {
			slog.Error("Failed to save tagged track to disk", "filePath", filePath, "error", err)
			continue
		}
		downloadedTracks = append(downloadedTracks, downloadedTrack)
		filePaths = append(filePaths, filePath)
		slog.Info("Track downloaded successfully", "title", downloadedTrack.Title, "filePath", filePath)
	}

	// Save local artwork file to album folder if enabled
	if len(downloadedTracks) > 0 && downloadedTracks[0].Album != nil {
		progressUpdater(90, "Saving album artwork...")
		if err := e.service.artworkService.SaveLocalArtwork(ctx, downloadedTracks[0], albumPath); err != nil {
			slog.Warn("Failed to save album artwork", "albumID", albumID, "error", err)
			// Don't fail the download for artwork issues
		}
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
