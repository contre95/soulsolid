package reorganize

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/contre95/soulsolid/src/music"
)

type ReorganizeJobTask struct {
	service *Service
}

func NewReorganizeJobTask(service *Service) *ReorganizeJobTask {
	return &ReorganizeJobTask{
		service: service,
	}
}

func (t *ReorganizeJobTask) MetadataKeys() []string {
	return []string{}
}

func (t *ReorganizeJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	fat32Safe := false
	if v, ok := job.Metadata["fat32_safe"]; ok {
		if b, ok := v.(bool); ok {
			fat32Safe = b
		}
	}

	totalTracks, err := t.service.library.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"moved":       0,
			"skipped":     0,
			"errors":      0,
			"msg":         "No tracks found in library",
		}, nil
	}

	job.Logger.Info("Starting file reorganization", "totalTracks", totalTracks, "fat32Safe", fat32Safe, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting reorganization of %d tracks", totalTracks))

	attempted := 0
	moved := 0
	skipped := 0
	errors := 0

	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			job.Logger.Info("File reorganization cancelled", "attempted", attempted, "moved", moved, "color", "orange")
			return nil, ctx.Err()
		default:
		}

		tracks, err := t.service.library.GetTracksPaginated(ctx, batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get tracks batch (offset %d): %w", offset, err)
		}

		for _, track := range tracks {
			select {
			case <-ctx.Done():
				job.Logger.Info("File reorganization cancelled", "attempted", attempted, "moved", moved, "color", "orange")
				return nil, ctx.Err()
			default:
			}

			progress := (attempted * 100) / totalTracks
			progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", attempted+1, totalTracks, track.Title))
			attempted++

			desiredPath, err := t.service.fileManager.GetLibraryPath(ctx, track)
			if err != nil {
				job.Logger.Warn("Failed to get desired path for track", "trackID", track.ID, "title", track.Title, "error", err, "color", "orange")
				errors++
				continue
			}

			if _, err := os.Stat(track.Path); os.IsNotExist(err) {
				job.Logger.Info("Skipping track with missing file", "trackID", track.ID, "title", track.Title, "path", track.Path, "color", "orange")
				skipped++
				continue
			}

			currentPath := filepath.Clean(track.Path)
			desiredPath = filepath.Clean(desiredPath)

			if fat32Safe {
				desiredPath = sanitizeFAT32Path(desiredPath)
				if currentPath != desiredPath {
					desiredPath = resolvePathConflict(desiredPath)
				}
			}

			if currentPath == desiredPath {
				job.Logger.Info("Track already in correct location", "trackID", track.ID, "title", track.Title, "path", currentPath, "color", "cyan")
				skipped++
				continue
			}

			job.Logger.Info("Moving track to new location", "trackID", track.ID, "title", track.Title, "from", currentPath, "to", desiredPath, "color", "yellow")
			newPath, err := t.service.fileManager.MoveTrackFile(ctx, track.Path, desiredPath)
			if err != nil {
				job.Logger.Warn("Failed to move track", "trackID", track.ID, "title", track.Title, "error", err, "color", "red")
				errors++
				continue
			}

			track.Path = newPath
			err = t.service.library.UpdateTrack(ctx, track)
			if err != nil {
				job.Logger.Warn("Failed to update track path in database", "trackID", track.ID, "title", track.Title, "newPath", newPath, "error", err, "color", "red")
				errors++
				continue
			}

			job.Logger.Info("Successfully moved track", "trackID", track.ID, "title", track.Title, "newPath", newPath, "color", "green")
			moved++
		}
	}

	finalMsg := fmt.Sprintf("Reorganization completed: %d path(s) modified, %d already correct, %d errors (of %d total tracks)", moved, skipped, errors, totalTracks)
	job.Logger.Info("File reorganization completed", "totalTracks", totalTracks, "moved", moved, "skipped", skipped, "errors", errors, "color", "green")
	progressUpdater(100, fmt.Sprintf("Done — %d path(s) modified, %d skipped, %d errors", moved, skipped, errors))

	return map[string]any{
		"totalTracks": totalTracks,
		"moved":       moved,
		"skipped":     skipped,
		"errors":      errors,
		"msg":         finalMsg,
	}, nil
}

func (t *ReorganizeJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up reorganization job", "jobID", job.ID)
	return nil
}
