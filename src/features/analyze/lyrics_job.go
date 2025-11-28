package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// LyricsJobTask handles lyrics analysis job execution
type LyricsJobTask struct {
	service *Service
}

// NewLyricsJobTask creates a new lyrics analysis job task
func NewLyricsJobTask(service *Service) *LyricsJobTask {
	return &LyricsJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for lyrics analysis jobs
func (t *LyricsJobTask) MetadataKeys() []string {
	return []string{}
}

// Execute performs the lyrics analysis operation
func (t *LyricsJobTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	job.Logger.Info("EXECUTE STARTED: Lyrics job task is running", "color", "pink")

	// Check if any lyrics providers are enabled
	enabledProviders := t.service.lyricsService.GetEnabledLyricsProviders()
	hasEnabledProviders := false
	for _, enabled := range enabledProviders {
		if enabled {
			hasEnabledProviders = true
			break
		}
	}

	if !hasEnabledProviders {
		job.Logger.Error("No lyrics providers are enabled, cannot proceed with lyrics analysis")
		return nil, fmt.Errorf("no lyrics providers are enabled - please enable at least one lyrics provider in the configuration")
	}

	job.Logger.Info("Enabled lyrics providers", "providers", enabledProviders, "color", "blue")

	// Get total track count for progress reporting
	totalTracks, err := t.service.libraryService.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"updated":     0,
		}, nil
	}

	job.Logger.Info("Starting lyrics analysis", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	skipped := 0

	// Process tracks in batches to avoid loading all into memory
	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			job.Logger.Info("Lyrics analysis cancelled", "processed", processed, "updated", updated)
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
				job.Logger.Info("Lyrics analysis cancelled", "processed", processed, "updated", updated)
				return nil, ctx.Err()
			default:
			}

			progress := (processed * 100) / totalTracks
			progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", processed+1, totalTracks, track.Title))

			// Skip tracks that already have lyrics
			if track.Metadata.Lyrics != "" {
				job.Logger.Info("Skipping track with existing lyrics", "trackID", track.ID, "title", track.Title, "lyricsLength", len(track.Metadata.Lyrics), "color", "orange")
				skipped++
				continue
			}

			// Try to fetch lyrics for this track
			job.Logger.Info("Fetching lyrics for track", "trackID", track.ID, "title", track.Title, "artist", track.Artists, "album", track.Album, "color", "cyan")
			err := t.service.lyricsService.AddLyricsWithBestProvider(ctx, track.ID)
			if err != nil {
				job.Logger.Warn("Failed to add lyrics for track, setting to [No Lyrics]", "trackID", track.ID, "title", track.Title, "error", err.Error(), "color", "orange")
				// Set lyrics to [No Lyrics] when fetching fails
				err = t.service.lyricsService.SetLyricsToNoLyrics(ctx, track.ID)
				if err != nil {
					job.Logger.Error("Failed to set [No Lyrics] for track", "trackID", track.ID, "title", track.Title, "error", err.Error())
				} else {
					updated++
					job.Logger.Info("Set [No Lyrics] for track", "trackID", track.ID, "title", track.Title, "color", "violet")
				}
				// Continue with other tracks - don't fail the entire job
			} else {
				updated++
				job.Logger.Info("Successfully added lyrics for track", "trackID", track.ID, "title", track.Title, "color", "green")
			}

			processed++
		}
	}

	job.Logger.Info("Lyrics analysis completed", "totalTracks", totalTracks, "processed", processed, "updated", updated, "skipped", skipped, "color", "green")

	// Create completion message for job tagging
	finalMessage := fmt.Sprintf("Lyrics analysis finished. Processed %d tracks (%d updated, %d skipped, %d errors).",
		totalTracks, updated, skipped, 0)
	job.Logger.Info(finalMessage)

	progressUpdater(100, fmt.Sprintf("Lyrics analysis completed - totalTracks=%d processed=%d updated=%d skipped=%d", totalTracks, processed, updated, skipped))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"updated":     updated,
		"skipped":     skipped,
		"errors":      0,
		"msg":         finalMessage,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *LyricsJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up lyrics analysis job", "jobID", job.ID)
	return nil
}
