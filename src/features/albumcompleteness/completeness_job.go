package albumcompleteness

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/contre95/soulsolid/src/music"
)

// albumBatchSize is how many albums are loaded from the library at a time.
const albumBatchSize = 100

// CompletenessParams is the metadata contract for an album completeness job.
type CompletenessParams struct {
	// Force re-checks albums that were already checked, and replaces manual counts.
	Force bool `json:"force"`
}

// CompletenessJobTask checks every album of the library against the metadata providers.
type CompletenessJobTask struct {
	service *Service
}

// NewCompletenessJobTask creates a new album completeness job task.
func NewCompletenessJobTask(service *Service) *CompletenessJobTask {
	return &CompletenessJobTask{service: service}
}

// Execute checks the albums one by one, throttled to respect the providers' rate limits.
func (t *CompletenessJobTask) Execute(ctx context.Context, job *music.Job, params CompletenessParams, progressUpdater func(int, string)) (map[string]any, error) {
	s := t.service
	filter := &music.AlbumFilter{Sort: music.AlbumSortTitle}
	total, err := s.repo.GetAlbumSummariesCount(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count albums: %w", err)
	}
	if total == 0 {
		return map[string]any{
			"totalAlbums": 0, "attempted": 0, "checked": 0, "incomplete": 0, "notFound": 0, "skipped": 0, "errors": 0,
			"msg": "Album completeness analysis finished. No albums found in library.",
		}, nil
	}

	job.Logger.Info("Starting album completeness analysis", "totalAlbums", total, "force", params.Force, "color", "blue")
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d albums", total))

	attempted, checked, incomplete, notFound, skipped, errorCount := 0, 0, 0, 0, 0, 0
	summary := func() map[string]any {
		return map[string]any{
			"totalAlbums": total, "attempted": attempted, "checked": checked, "incomplete": incomplete,
			"notFound": notFound, "skipped": skipped, "errors": errorCount,
		}
	}

	for offset := 0; offset < total; offset += albumBatchSize {
		summaries, err := s.repo.GetAlbumSummaries(ctx, albumBatchSize, offset, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get albums batch (offset %d): %w", offset, err)
		}
		for _, sum := range summaries {
			if ctx.Err() != nil {
				job.Logger.Info("Album completeness analysis cancelled", "attempted", attempted)
				return nil, ctx.Err()
			}
			attempted++
			progressUpdater(attempted*100/total, fmt.Sprintf("Checking album %d/%d: %s", attempted, total, sum.Title))

			album, err := s.repo.GetAlbum(ctx, sum.ID)
			if err != nil || album == nil {
				job.Logger.Warn("Failed to load album", "albumID", sum.ID, "title", sum.Title, "error", err, "color", "orange")
				errorCount++
				continue
			}
			if !params.Force && (album.Attributes[music.AlbumAttrCompletenessCheckedAt] != "" ||
				album.Attributes[music.AlbumAttrCompletenessSource] == music.CompletenessSourceManual) {
				skipped++
				continue
			}

			result, err := s.checkAlbum(ctx, album, sum.TrackCount)
			switch {
			case errors.Is(err, ErrNoMatch):
				notFound++
				job.Logger.Info("No release found for album", "title", sum.Title, "artists", sum.Artists, "color", "yellow")
			case err != nil:
				errorCount++
				job.Logger.Warn("Failed to check album", "title", sum.Title, "artists", sum.Artists, "error", err, "color", "orange")
			default:
				checked++
				if result.Status == StatusIncomplete {
					incomplete++
					job.Logger.Info("Album is missing tracks", "title", sum.Title, "artists", sum.Artists,
						"have", result.LocalCount, "expected", result.ExpectedCount, "color", "cyan")
				}
			}

			// Wait out the providers' rate limit before the next lookup.
			select {
			case <-ctx.Done():
				job.Logger.Info("Album completeness analysis cancelled", "attempted", attempted)
				return nil, ctx.Err()
			case <-time.After(s.throttle):
			}
		}
	}

	msg := fmt.Sprintf("Album completeness analysis finished. Checked %d albums: %d incomplete, %d not found, %d skipped, %d errors.",
		checked+notFound, incomplete, notFound, skipped, errorCount)
	job.Logger.Info(msg, "color", "green")
	progressUpdater(100, fmt.Sprintf("Analysis completed - %d incomplete albums", incomplete))
	stats := summary()
	stats["msg"] = msg
	return stats, nil
}

// Cleanup is a no-op for album completeness jobs.
func (t *CompletenessJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up album completeness job", "jobID", job.ID)
	return nil
}
