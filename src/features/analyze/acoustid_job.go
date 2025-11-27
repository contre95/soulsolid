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
	tracks, err := t.service.libraryService.GetTracks(ctx) // TODO: loading all tracks into memory could be problematic for large music libraries
	// 1. **Use pagination**: Replace `GetTracks` with `GetTracksPaginated` to process tracks in batches
	// 2. **Channel-based streaming**: Modify the interface to return a channel that yields tracks incrementally
	// 3. **Iterator pattern**: Implement an iterator that fetches tracks on-demand
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks: %w", err)
	}

	totalTracks := len(tracks)
	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"updated":     0,
		}, nil
	}

	job.Logger.Info("Starting AcoustID analysis", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	skipped := 0

	for i, track := range tracks {
		select {
		case <-ctx.Done():
			job.Logger.Info("AcoustID analysis cancelled", "processed", processed, "updated", updated)
			return nil, ctx.Err()
		default:
		}

		progress := (i * 100) / totalTracks
		progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", i+1, totalTracks, track.Title))

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
			job.Logger.Warn("Failed to add AcoustID for track", "trackID", track.ID, "title", track.Title, "error", err, "color", "orange")
			// Continue with other tracks - don't fail the entire job
		} else {
			updated++
			job.Logger.Info("Successfully added AcoustID for track", "trackID", track.ID, "title", track.Title, "color", "green")
		}

		processed++
	}

	job.Logger.Info("AcoustID analysis completed", "totalTracks", totalTracks, "processed", processed, "updated", updated, "skipped", skipped, "color", "green")
	progressUpdater(100, fmt.Sprintf("Analysis completed - %d updated, %d skipped", updated, skipped))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"updated":     updated,
		"skipped":     skipped,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *AcoustIDJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up AcoustID analysis job", "jobID", job.ID)
	return nil
}
