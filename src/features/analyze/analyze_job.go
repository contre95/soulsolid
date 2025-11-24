package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// AnalyzeJobTask handles analysis job execution
type AnalyzeJobTask struct {
	service *Service
}

// NewAnalyzeJobTask creates a new analyze job task
func NewAnalyzeJobTask(service *Service) *AnalyzeJobTask {
	return &AnalyzeJobTask{
		service: service,
	}
}

// MetadataKeys returns the required metadata keys for analyze jobs
func (t *AnalyzeJobTask) MetadataKeys() []string {
	return []string{}
}

// Execute performs the analysis operation
func (t *AnalyzeJobTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	switch job.Type {
	case "analyze_acoustid":
		return t.executeAcoustIDAnalysis(ctx, job, progressUpdater)
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", job.Type)
	}
}

// executeAcoustIDAnalysis handles AcoustID analysis jobs
func (t *AnalyzeJobTask) executeAcoustIDAnalysis(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	slog.Info("Starting AcoustID analysis", "jobID", job.ID)

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

	slog.Info("Starting AcoustID analysis", "totalTracks", totalTracks)
	progressUpdater(0, fmt.Sprintf("Starting analysis of %d tracks", totalTracks))

	processed := 0
	updated := 0
	skipped := 0

	for i, track := range tracks {
		select {
		case <-ctx.Done():
			slog.Info("AcoustID analysis cancelled", "processed", processed, "updated", updated)
			return nil, ctx.Err()
		default:
		}

		progress := (i * 100) / totalTracks
		progressUpdater(progress, fmt.Sprintf("Processing track %d/%d: %s", i+1, totalTracks, track.Title))

		// Skip tracks that already have AcoustID
		if track.AcoustID != "" {
			slog.Debug("Skipping track with existing AcoustID", "trackID", track.ID, "acoustID", track.AcoustID)
			skipped++
			continue
		}

		// Call the existing AddChromaprintAndAcoustID method
		err := t.service.taggingService.AddChromaprintAndAcoustID(ctx, track.ID)
		if err != nil {
			slog.Warn("Failed to add AcoustID for track", "trackID", track.ID, "title", track.Title, "error", err)
			// Continue with other tracks - don't fail the entire job
		} else {
			updated++
			slog.Info("Successfully added AcoustID for track", "trackID", track.ID, "title", track.Title)
		}

		processed++
	}

	slog.Info("AcoustID analysis completed", "totalTracks", totalTracks, "processed", processed, "updated", updated, "skipped", skipped)
	progressUpdater(100, fmt.Sprintf("Analysis completed - %d updated, %d skipped", updated, skipped))

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   processed,
		"updated":     updated,
		"skipped":     skipped,
	}, nil
}

// Cleanup performs cleanup after job completion
func (t *AnalyzeJobTask) Cleanup(job *jobs.Job) error {
	slog.Debug("Cleaning up analyze job", "jobID", job.ID)
	return nil
}
