package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// AcoustIDJobTask handles AcoustID analysis job execution
type AcoustIDJobTask struct {
	service *Service
}

// NewAcoustIDJobTask creates a new AcoustID analysis job task
func NewAcoustIDJobTask(service *Service) *AcoustIDJobTask {
	return &AcoustIDJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for AcoustID analysis jobs
func (t *AcoustIDJobTask) MetadataKeys() []string {
	return []string{}
}

// Execute performs the AcoustID analysis operation
func (t *AcoustIDJobTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	// Get total track count for progress reporting
	totalTracks, err := t.service.libraryService.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks":       0,
			"processed":         0,
			"acoustidsAdded":    0,
			"fingerprintsAdded": 0,
			"skipped":           0,
		}, nil
	}

	job.Logger.Info("Starting AcoustID analysis", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	fingerprintsAdded := 0
	skipped := 0

	// Process tracks in batches to avoid loading all into memory
	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			job.Logger.Info("AcoustID analysis cancelled", "processed", processed, "acoustidsAdded", updated, "fingerprintsAdded", fingerprintsAdded)
			return nil, ctx.Err()
		default:
		}

		// Get next batch of tracks
		tracks, err := t.service.libraryService.GetTracksPaginated(ctx, batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get tracks batch (offset %d): %w", offset, err)
		}

		for _, track := range tracks {
			select {
			case <-ctx.Done():
				job.Logger.Info("AcoustID analysis cancelled", "processed", processed, "acoustidsAdded", updated, "fingerprintsAdded", fingerprintsAdded)
				return nil, ctx.Err()
			default:
			}

			progress := (processed * 100) / totalTracks
			progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", processed+1, totalTracks, track.Title))

			// Skip tracks that already have AcoustID
			acoustID := ""
			if track.Attributes != nil {
				acoustID = track.Attributes["acoustid"]
			}
			if acoustID != "" {
				job.Logger.Info("Skipping track with existing AcoustID", "trackID", track.ID, "title", track.Title, "acoustID", acoustID, "color", "orange")
				skipped++
				continue
			}

			// Call the existing AddChromaprintAndAcoustID method
			job.Logger.Info("Analyzing track fingerprint", "trackID", track.ID, "title", track.Title, "artist", track.Artists, "color", "cyan")
			err := t.service.taggingService.AddChromaprintAndAcoustID(ctx, track.ID)
			if err != nil {
				job.Logger.Warn("Failed to add fingerprint and AcoustID for track", "trackID", track.ID, "title", track.Title, "error", err, "color", "orange")
				// Continue with other tracks - don't fail the entire job
			} else {
				// Check if AcoustID was actually added
				updatedTrack, err := t.service.libraryService.GetTrack(ctx, track.ID)
				if err != nil {
					job.Logger.Warn("Failed to verify AcoustID addition for track", "trackID", track.ID, "title", track.Title, "error", err, "color", "orange")
					fingerprintsAdded++ // Assume fingerprint was added
				} else {
					acoustID := ""
					if updatedTrack.Attributes != nil {
						acoustID = updatedTrack.Attributes["acoustid"]
					}
					if acoustID != "" {
						updated++
						job.Logger.Info("Successfully added AcoustID for track", "trackID", track.ID, "title", track.Title, "color", "green")
					} else {
						fingerprintsAdded++
						job.Logger.Info("Added fingerprint for track, AcoustID lookup failed or not configured", "trackID", track.ID, "title", track.Title, "color", "yellow")
					}
				}
			}

			processed++
		}
	}

	job.Logger.Info("AcoustID analysis completed", "totalTracks", totalTracks, "processed", processed, "acoustidsAdded", updated, "fingerprintsAdded", fingerprintsAdded, "skipped", skipped, "color", "green")
	progressUpdater(100, fmt.Sprintf("Analysis completed - %d AcoustIDs added, %d fingerprints added, %d skipped", updated, fingerprintsAdded, skipped))

	return map[string]any{
		"totalTracks":       totalTracks,
		"processed":         processed,
		"acoustidsAdded":    updated,
		"fingerprintsAdded": fingerprintsAdded,
		"skipped":           skipped,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *AcoustIDJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up AcoustID analysis job", "jobID", job.ID)
	return nil
}
