package analyze

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// ReorganizeJobTask handles file reorganization job execution
type ReorganizeJobTask struct {
	service *Service
}

// NewReorganizeJobTask creates a new reorganization job task
func NewReorganizeJobTask(service *Service) *ReorganizeJobTask {
	return &ReorganizeJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for reorganization jobs
func (t *ReorganizeJobTask) MetadataKeys() []string {
	return []string{}
}

// Execute performs the file reorganization operation
func (t *ReorganizeJobTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	// Get total track count for progress reporting
	totalTracks, err := t.service.library.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"moved":       0,
			"skipped":     0,
			"errors":      0,
		}, nil
	}

	job.Logger.Info("Starting file reorganization", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting reorganization of %d tracks", totalTracks))

	processed := 0
	moved := 0
	skipped := 0
	errors := 0

	// Process tracks in batches to avoid loading all into memory
	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			job.Logger.Info("File reorganization cancelled", "processed", processed, "moved", moved, "color", "orange")
			return nil, ctx.Err()
		default:
		}

		// Get next batch of tracks
		tracks, err := t.service.library.GetTracksPaginated(ctx, batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get tracks batch (offset %d): %w", offset, err)
		}

		for _, track := range tracks {
			select {
			case <-ctx.Done():
				job.Logger.Info("File reorganization cancelled", "processed", processed, "moved", moved, "color", "orange")
				return nil, ctx.Err()
			default:
			}

			progress := (processed * 100) / totalTracks
			progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", processed+1, totalTracks, track.Title))

			// Get the desired path for this track based on current config
			desiredPath, err := t.service.fileManager.GetLibraryPath(ctx, track)
			if err != nil {
				job.Logger.Warn("Failed to get desired path for track", "trackID", track.ID, "title", track.Title, "error", err, "color", "orange")
				errors++
				continue
			}

			// Check if the track file exists on disk
			if _, err := os.Stat(track.Path); os.IsNotExist(err) {
				job.Logger.Info("Skipping track with missing file", "trackID", track.ID, "title", track.Title, "path", track.Path, "color", "orange")
				skipped++
				continue
			}

			// Normalize paths for comparison (handle relative vs absolute paths)
			currentPath := filepath.Clean(track.Path)
			desiredPath = filepath.Clean(desiredPath)

			// Check if the track is already in the correct location
			if currentPath == desiredPath {
				job.Logger.Info("Track already in correct location", "trackID", track.ID, "title", track.Title, "path", currentPath, "color", "cyan")
				skipped++
				continue
			}

			// Move the track to the new location
			job.Logger.Info("Moving track to new location", "trackID", track.ID, "title", track.Title, "from", currentPath, "to", desiredPath, "color", "yellow")
			newPath, err := t.service.fileManager.MoveTrack(ctx, track)
			if err != nil {
				job.Logger.Warn("Failed to move track", "trackID", track.ID, "title", track.Title, "error", err, "color", "red")
				errors++
				continue
			}

			// Update the track path in the database
			track.Path = newPath
			err = t.service.library.UpdateTrack(ctx, track)
			if err != nil {
				job.Logger.Warn("Failed to update track path in database", "trackID", track.ID, "title", track.Title, "newPath", newPath, "error", err, "color", "red")
				errors++
				continue
			}

			job.Logger.Info("Successfully moved track", "trackID", track.ID, "title", track.Title, "newPath", newPath, "color", "green")
			moved++
			processed++
		}
	}

	job.Logger.Info("File reorganization completed", "totalTracks", totalTracks, "processed", processed, "moved", moved, "skipped", skipped, "errors", errors, "color", "green")
	progressUpdater(100, fmt.Sprintf("Reorganization completed - moved %d, skipped %d, errors %d", moved, skipped, errors))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"moved":       moved,
		"skipped":     skipped,
		"errors":      errors,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *ReorganizeJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up reorganization job", "jobID", job.ID)
	return nil
}
