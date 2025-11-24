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
	slog.Info("Starting lyrics analysis", "jobID", job.ID)

	// Check if any lyrics providers are enabled
	enabledProviders := t.service.taggingService.GetEnabledLyricsProviders()
	hasEnabledProviders := false
	for _, enabled := range enabledProviders {
		if enabled {
			hasEnabledProviders = true
			break
		}
	}

	if !hasEnabledProviders {
		slog.Error("No lyrics providers are enabled, cannot proceed with lyrics analysis")
		return nil, fmt.Errorf("no lyrics providers are enabled - please enable at least one lyrics provider in the configuration")
	}

	slog.Info("Enabled lyrics providers", "providers", enabledProviders)

	// Get all tracks from library
	tracks, err := t.service.libraryService.GetTracks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks: %w", err)
	}

	totalTracks := len(tracks)
	if totalTracks == 0 {
		slog.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"updated":     0,
		}, nil
	}

	slog.Info("Starting lyrics analysis", "totalTracks", totalTracks)
	progressUpdater(0, fmt.Sprintf("Starting lyrics analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	skipped := 0

	for i, track := range tracks {
		select {
		case <-ctx.Done():
			slog.Info("Lyrics analysis cancelled", "processed", processed, "updated", updated)
			return nil, ctx.Err()
		default:
		}

		progress := (i * 100) / totalTracks
		progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", i+1, totalTracks, track.Title))

		// Skip tracks that already have lyrics
		if track.Metadata.Lyrics != "" {
			slog.Debug("Skipping track with existing lyrics", "trackID", track.ID, "lyricsLength", len(track.Metadata.Lyrics))
			skipped++
			continue
		}

		// Try to fetch lyrics for this track
		slog.Info("Fetching lyrics for track", "trackID", track.ID, "title", track.Title, "artist", track.Artists, "album", track.Album)
		err := t.service.taggingService.AddLyrics(ctx, track.ID)
		if err != nil {
			slog.Warn("Failed to add lyrics for track, setting to [No Lyrics]", "trackID", track.ID, "title", track.Title, "error", err.Error())
			// Set lyrics to [No Lyrics] when fetching fails
			err = t.service.taggingService.SetLyricsToNoLyrics(ctx, track.ID)
			if err != nil {
				slog.Error("Failed to set [No Lyrics] for track", "trackID", track.ID, "title", track.Title, "error", err.Error())
			} else {
				updated++
				slog.Info("Set [No Lyrics] for track", "trackID", track.ID, "title", track.Title)
			}
			// Continue with other tracks - don't fail the entire job
		} else {
			updated++
			slog.Info("Successfully added lyrics for track", "trackID", track.ID, "title", track.Title)
		}

		processed++
	}

	slog.Info("Lyrics analysis completed", "totalTracks", totalTracks, "processed", processed, "updated", updated, "skipped", skipped)
	progressUpdater(100, fmt.Sprintf("Analysis completed - %d updated, %d skipped", updated, skipped))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"updated":     updated,
		"skipped":     skipped,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *LyricsJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up lyrics analysis job", "jobID", job.ID)
	return nil
}
