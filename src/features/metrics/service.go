package metrics

import (
	"context"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service provides metrics functionality for the music library.
type Service struct {
	metrics       music.LibraryMetrics
	configManager *config.Manager
}

// NewService creates a new metrics service.
func NewService(metrics music.LibraryMetrics, cfgManager *config.Manager) *Service {
	return &Service{
		metrics:       metrics,
		configManager: cfgManager,
	}
}

// Metric represents a single metric data point.
type Metric struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Value int    `json:"value"`
}

// MetricsData holds all metrics for display.
type MetricsData struct {
	GenreCounts          []Metric `json:"genre_counts"`
	LyricsStats          []Metric `json:"lyrics_stats"`
	MetadataCompleteness []Metric `json:"metadata_completeness"`
	FormatDistribution   []Metric `json:"format_distribution"`
	YearDistribution     []Metric `json:"year_distribution"`
	TotalTracks          int      `json:"total_tracks"`
	TotalArtists         int      `json:"total_artists"`
	TotalAlbums          int      `json:"total_albums"`
}

// GetAllMetrics retrieves all stored metrics from the database.
func (s *Service) GetAllMetrics(ctx context.Context) (*MetricsData, error) {
	data := &MetricsData{}

	// Get basic counts using LibraryMetrics interface
	if totalTracks, err := s.metrics.GetTotalTracks(ctx); err != nil {
		slog.Warn("Failed to get track count", "error", err)
	} else {
		data.TotalTracks = totalTracks
	}
	if totalArtists, err := s.metrics.GetTotalArtists(ctx); err != nil {
		slog.Warn("Failed to get artist count", "error", err)
	} else {
		data.TotalArtists = totalArtists
	}
	if totalAlbums, err := s.metrics.GetTotalAlbums(ctx); err != nil {
		slog.Warn("Failed to get album count", "error", err)
	} else {
		data.TotalAlbums = totalAlbums
	}

	// Get stored metrics
	var err error
	data.GenreCounts, err = s.getMetricsByType(ctx, "genre_counts")
	if err != nil {
		slog.Warn("Failed to get genre counts", "error", err)
	}

	data.LyricsStats, err = s.getMetricsByType(ctx, "lyrics_stats")
	if err != nil {
		slog.Warn("Failed to get lyrics stats", "error", err)
	}

	data.MetadataCompleteness, err = s.getMetricsByType(ctx, "metadata_completeness")
	if err != nil {
		slog.Warn("Failed to get metadata completeness", "error", err)
	}

	data.FormatDistribution, err = s.getMetricsByType(ctx, "format_distribution")
	if err != nil {
		slog.Warn("Failed to get format distribution", "error", err)
	}

	data.YearDistribution, err = s.getMetricsByType(ctx, "year_distribution")
	if err != nil {
		slog.Warn("Failed to get year distribution", "error", err)
	}

	return data, nil
}

// convertMapToMetrics converts a map[string]int to []Metric
func convertMapToMetrics(data map[string]int, metricType string) []Metric {
	metrics := make([]Metric, 0, len(data))
	for key, value := range data {
		metrics = append(metrics, Metric{
			Type:  metricType,
			Key:   key,
			Value: value,
		})
	}
	return metrics
}

// getMetricsByType retrieves metrics of a specific type.
func (s *Service) getMetricsByType(ctx context.Context, metricType string) ([]Metric, error) {
	stored, err := s.metrics.GetStoredMetrics(ctx, metricType)
	if err != nil {
		return nil, err
	}

	metrics := make([]Metric, len(stored))
	for i, s := range stored {
		metrics[i] = Metric{Type: s.Type, Key: s.Key, Value: s.Value}
	}
	return metrics, nil
}
