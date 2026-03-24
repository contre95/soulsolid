package lyrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
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
func (t *LyricsJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	job.Logger.Info("EXECUTE STARTED: Lyrics job task is running", "color", "pink")

	// Check if any lyrics providers are enabled
	enabledProviders := t.service.GetEnabledLyricsProviders()
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

	// Get initial queue state to track new items added during this job
	initialQueueItems := t.service.GetLyricsQueueItems()
	initialQueueIDs := make(map[string]bool)
	for id := range initialQueueItems {
		initialQueueIDs[id] = true
	}

	// Get total track count for progress reporting
	totalTracks, err := t.service.libraryRepo.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}

	if totalTracks == 0 {
		job.Logger.Info("No tracks found in library")
		return map[string]any{
			"totalTracks": 0,
			"processed":   0,
			"updated":     0,
			"queued":      0,
		}, nil
	}

	job.Logger.Info("Starting lyrics analysis", "totalTracks", totalTracks, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	skipped := 0
	errors := 0

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
		tracks, err := t.service.libraryRepo.GetTracksPaginated(ctx, batchSize, offset)
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

			job.Logger.Debug("Processing track", "trackID", track.ID, "title", track.Title, "hasLyrics", track.HasLyrics, "lyricsLength", len(track.Metadata.Lyrics))

			// Skip tracks explicitly marked as having no lyrics (has_lyrics = false and no lyrics)
			if !track.HasLyrics {
				job.Logger.Info("Track explicitly marked as having no lyrics - skipping", "trackID", track.ID, "title", track.Title, "has_lyrics", track.HasLyrics)
				skipped++
				processed++
				if track.Metadata.Lyrics != "" {
					job.Logger.Warn("Track has lyrics but marked as without lyrics - fix inconsistency", "trackID", track.ID, "title", track.Title, "lyricsLength", len(track.Metadata.Lyrics))
				}
				continue
			}

			// Get the specified provider from job metadata
			provider, ok := job.Metadata["provider"].(string)
			if !ok || provider == "" {
				job.Logger.Error("No provider specified in job metadata")
				return nil, fmt.Errorf("no lyrics provider specified in job metadata")
			}

			// Try to fetch lyrics for this track using the specified provider
			job.Logger.Info("Fetching lyrics for track", "trackID", track.ID, "title", track.Title, "artist", track.Artists, "album", track.Album, "provider", provider, "color", "cyan")
			result, err := t.service.AddLyrics(ctx, track.ID, provider)
			if err != nil {
				job.Logger.Error("Failed to add lyrics for track", "trackID", track.ID, "title", track.Title, "provider", provider, "error", err.Error(), "manual_fix", "<a href='/ui/library/tag/edit/"+track.ID+"' target='_blank'>track</a>")
				errors++
				// Continue with other tracks - don't fail the entire job
			} else {
				job.Logger.Debug("AddLyrics result", "trackID", track.ID, "result", int(result))
				switch result {
				case LyricsAdded:
					updated++
					job.Logger.Info("Added lyrics for track", "trackID", track.ID, "title", track.Title, "provider", provider, "color", "green")
				case LyricsQueued:
					job.Logger.Info("Queued lyrics for track (differ from existing)", "trackID", track.ID, "title", track.Title, "provider", provider, "color", "blue")
				case LyricsSkipped:
					skipped++
					job.Logger.Info("Skipped lyrics for track (identical to existing)", "trackID", track.ID, "title", track.Title, "provider", provider, "color", "yellow")
				}
			}
			processed++
		}
	}

	// Count new queue items added during this job
	finalQueueItems := t.service.GetLyricsQueueItems()
	existingLyricsQueued := 0
	lyric404Queued := 0
	failedLyricsQueued := 0

	for id, item := range finalQueueItems {
		if !initialQueueIDs[id] {
			switch item.Type {
			case ExistingLyrics:
				existingLyricsQueued++
				job.Logger.Info("New existing_lyrics queue item added", "trackID", id, "color", "cyan")
			case Lyric404:
				lyric404Queued++
				job.Logger.Info("New lyric_404 queue item added", "trackID", id, "color", "cyan")
			case FailedLyrics:
				failedLyricsQueued++
				job.Logger.Info("New failed_lyrics queue item added", "trackID", id, "color", "cyan")
			}
		}
	}
	queued := existingLyricsQueued + lyric404Queued + failedLyricsQueued
	job.Logger.Info("Lyrics analysis completed", "totalTracks", totalTracks, "processed", processed, "updated", updated, "skipped", skipped, "queued", queued, "color", "green")

	job.Logger.Debug("Final counters", "totalTracks", totalTracks, "updated", updated, "skipped", skipped, "errors", errors, "existingQueued", existingLyricsQueued, "404Queued", lyric404Queued, "failedQueued", failedLyricsQueued)

	queueSummary := ""
	if existingLyricsQueued > 0 || lyric404Queued > 0 || failedLyricsQueued > 0 {
		queueSummary = fmt.Sprintf(" [Queue: %d existing_lyrics, %d lyric_404, %d failed_lyrics]", existingLyricsQueued, lyric404Queued, failedLyricsQueued)
	}
	finalMessage := fmt.Sprintf("Lyrics analysis finished. Processed %d tracks (%d updated, %d skipped, %d queued).%s",
		totalTracks, updated, skipped, queued, queueSummary)
	job.Logger.Info(finalMessage)

	progressUpdater(100, fmt.Sprintf("Lyrics analysis completed - totalTracks=%d processed=%d updated=%d skipped=%d errors=%d", totalTracks, processed, updated, skipped, errors))

	return map[string]any{
		"totalTracks":          totalTracks,
		"processed":            processed,
		"updated":              updated,
		"skipped":              skipped,
		"errors":               errors,
		"existingLyricsQueued": existingLyricsQueued,
		"lyric404Queued":       lyric404Queued,
		"failedLyricsQueued":   failedLyricsQueued,
		"msg":                  finalMessage,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *LyricsJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up lyrics analysis job", "jobID", job.ID)
	return nil
}
