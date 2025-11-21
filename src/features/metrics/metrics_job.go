package metrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/jobs"
)

// MetricsCalculationTask implements jobs.Task for calculating library metrics.
type MetricsCalculationTask struct {
	metrics LibraryMetrics
}

// NewMetricsCalculationTask creates a new metrics calculation task.
func NewMetricsCalculationTask(metrics LibraryMetrics) *MetricsCalculationTask {
	return &MetricsCalculationTask{
		metrics: metrics,
	}
}

// MetadataKeys returns the required metadata keys (none needed).
func (t *MetricsCalculationTask) MetadataKeys() []string {
	return []string{}
}

// Execute runs the metrics calculation logic.
func (t *MetricsCalculationTask) Execute(ctx context.Context, job *jobs.Job, progressUpdater func(int, string)) (map[string]any, error) {
	slog.Info("Starting metrics calculation")

	// Clear existing metrics
	if err := t.clearMetrics(ctx); err != nil {
		return nil, fmt.Errorf("failed to clear metrics: %w", err)
	}

	progressUpdater(10, "Cleared existing metrics")

	// Calculate and store genre distribution
	if err := t.calculateAndStoreGenreCounts(ctx, progressUpdater); err != nil {
		return nil, fmt.Errorf("failed to calculate genre counts: %w", err)
	}

	// Calculate and store lyrics statistics
	if err := t.calculateAndStoreLyricsStats(ctx, progressUpdater); err != nil {
		return nil, fmt.Errorf("failed to calculate lyrics stats: %w", err)
	}

	// Calculate and store metadata completeness
	if err := t.calculateAndStoreMetadataCompleteness(ctx, progressUpdater); err != nil {
		return nil, fmt.Errorf("failed to calculate metadata completeness: %w", err)
	}

	// Calculate and store format distribution
	if err := t.calculateAndStoreFormatDistribution(ctx, progressUpdater); err != nil {
		return nil, fmt.Errorf("failed to calculate format distribution: %w", err)
	}

	// Calculate and store year distribution
	if err := t.calculateAndStoreYearDistribution(ctx, progressUpdater); err != nil {
		return nil, fmt.Errorf("failed to calculate year distribution: %w", err)
	}

	progressUpdater(100, "Metrics calculation completed")
	slog.Info("Metrics calculation completed successfully")

	return map[string]any{"status": "completed"}, nil
}

// Cleanup performs cleanup after job execution.
func (t *MetricsCalculationTask) Cleanup(job *jobs.Job) error {
	// No cleanup needed
	return nil
}

// clearMetrics removes all existing metrics.
func (t *MetricsCalculationTask) clearMetrics(ctx context.Context) error {
	return t.metrics.ClearStoredMetrics(ctx)
}

// calculateAndStoreGenreCounts calculates and stores genre distribution.
func (t *MetricsCalculationTask) calculateAndStoreGenreCounts(ctx context.Context, progressUpdater func(int, string)) error {
	progressUpdater(20, "Calculating genre distribution")

	genreDist, err := t.metrics.GetGenreDistribution(ctx)
	if err != nil {
		return err
	}

	for genre, count := range genreDist {
		if err := t.storeMetric(ctx, "genre_counts", genre, count); err != nil {
			return err
		}
	}

	return nil
}

// calculateAndStoreLyricsStats calculates and stores lyrics statistics.
func (t *MetricsCalculationTask) calculateAndStoreLyricsStats(ctx context.Context, progressUpdater func(int, string)) error {
	progressUpdater(40, "Analyzing lyrics presence")

	lyricsStats, err := t.metrics.GetLyricsStats(ctx)
	if err != nil {
		return err
	}

	if err := t.storeMetric(ctx, "lyrics_stats", "has_lyrics", lyricsStats.WithLyrics); err != nil {
		return err
	}
	if err := t.storeMetric(ctx, "lyrics_stats", "no_lyrics", lyricsStats.WithoutLyrics); err != nil {
		return err
	}

	return nil
}

// calculateAndStoreMetadataCompleteness calculates and stores metadata completeness.
func (t *MetricsCalculationTask) calculateAndStoreMetadataCompleteness(ctx context.Context, progressUpdater func(int, string)) error {
	progressUpdater(60, "Checking metadata completeness")

	metadataStats, err := t.metrics.GetMetadataCompleteness(ctx)
	if err != nil {
		return err
	}

	if err := t.storeMetric(ctx, "metadata_completeness", "complete", metadataStats.Complete); err != nil {
		return err
	}
	if err := t.storeMetric(ctx, "metadata_completeness", "missing_genre", metadataStats.MissingGenre); err != nil {
		return err
	}
	if err := t.storeMetric(ctx, "metadata_completeness", "missing_year", metadataStats.MissingYear); err != nil {
		return err
	}
	if err := t.storeMetric(ctx, "metadata_completeness", "missing_lyrics", metadataStats.MissingLyrics); err != nil {
		return err
	}

	return nil
}

// calculateAndStoreFormatDistribution calculates and stores format distribution.
func (t *MetricsCalculationTask) calculateAndStoreFormatDistribution(ctx context.Context, progressUpdater func(int, string)) error {
	progressUpdater(80, "Analyzing audio formats")

	formatDist, err := t.metrics.GetFormatDistribution(ctx)
	if err != nil {
		return err
	}

	for format, count := range formatDist {
		if err := t.storeMetric(ctx, "format_distribution", format, count); err != nil {
			return err
		}
	}

	return nil
}

// calculateAndStoreYearDistribution calculates and stores year distribution.
func (t *MetricsCalculationTask) calculateAndStoreYearDistribution(ctx context.Context, progressUpdater func(int, string)) error {
	progressUpdater(95, "Calculating temporal metrics")

	yearDist, err := t.metrics.GetYearDistribution(ctx)
	if err != nil {
		return err
	}

	for year, count := range yearDist {
		if err := t.storeMetric(ctx, "year_distribution", year, count); err != nil {
			return err
		}
	}

	return nil
}

// storeMetric stores a metric in the database.
func (t *MetricsCalculationTask) storeMetric(ctx context.Context, metricType, key string, value int) error {
	return t.metrics.StoreMetric(ctx, metricType, key, value)
}
