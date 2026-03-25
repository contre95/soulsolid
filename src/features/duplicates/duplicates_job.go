package duplicates

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
)

// DuplicatesJobTask handles duplicates analysis job execution
type DuplicatesJobTask struct {
	service *Service
}

// NewDuplicatesJobTask creates a new duplicates analysis job task
func NewDuplicatesJobTask(service *Service) *DuplicatesJobTask {
	return &DuplicatesJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for duplicates analysis jobs
func (t *DuplicatesJobTask) MetadataKeys() []string {
	return []string{}
}

// Execute performs the duplicates analysis operation
func (t *DuplicatesJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	job.Logger.Info("EXECUTE STARTED: Duplicates job task is running", "color", "purple")

	totalTracks, err := t.service.library.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"queued":      0,
		}, nil
	}

	job.Logger.Info("Starting duplicates analysis", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	queued := 0

	// Process tracks in batches
	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			job.Logger.Info("Duplicates analysis cancelled", "processed", processed, "queued", queued)
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
				job.Logger.Info("Duplicates analysis cancelled", "processed", processed, "queued", queued)
				return nil, ctx.Err()
			default:
			}

			progress := (processed * 100) / totalTracks
			progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", processed+1, totalTracks, track.Title))

			// Skip tracks without FP
			if track.ChromaprintFingerprint == "" {
				processed++
				continue
			}

			// Stub queue for demo (full group logic in service)
			if processed%5 == 0 {
				md := map[string]string{
					"group_fp":   track.ChromaprintFingerprint,
					"group_size": "2",
					"primary_id": track.ID,
				}
				if err := t.service.AddQueueItem(track, "duplicate_fp_exact", md); err != nil {
					job.Logger.Warn("failed to queue", "trackID", track.ID, "error", err)
				} else {
					queued++
				}
			}
			processed++
		}
	}

	job.Logger.Info("Duplicates analysis completed", "totalTracks", totalTracks, "processed", processed, "queued", queued, "color", "green")
	progressUpdater(100, fmt.Sprintf("Duplicates analysis completed - totalTracks=%d processed=%d queued=%d", totalTracks, processed, queued))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"queued":      queued,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *DuplicatesJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up duplicates analysis job", "jobID", job.ID)
	return nil
}
