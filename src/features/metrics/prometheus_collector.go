package metrics

import (
	"context"
	"log/slog"
	"os"

	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector implements prometheus.Collector for soulsolid metrics.
type PrometheusCollector struct {
	libraryMetrics LibraryMetrics
	jobService     jobs.JobService

	// Library metrics
	tracksTotal           prometheus.Gauge
	artistsTotal          prometheus.Gauge
	albumsTotal           prometheus.Gauge
	genreCount            *prometheus.GaugeVec
	formatCount           *prometheus.GaugeVec
	yearCount             *prometheus.GaugeVec
	metadataComplete      prometheus.Gauge
	metadataMissingGenre  prometheus.Gauge
	metadataMissingYear   prometheus.Gauge
	metadataMissingLyrics prometheus.Gauge
	lyricsPresent         prometheus.Gauge
	lyricsMissing         prometheus.Gauge

	// Job metrics
	jobsTotal *prometheus.GaugeVec

	// Build info
	buildInfo *prometheus.GaugeVec
}

// NewPrometheusCollector creates a new PrometheusCollector.
func NewPrometheusCollector(libraryMetrics LibraryMetrics, jobService jobs.JobService) *PrometheusCollector {
	const namespace = "soulsolid"

	return &PrometheusCollector{
		libraryMetrics: libraryMetrics,
		jobService:     jobService,

		tracksTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_tracks_total",
			Help:      "Total number of tracks in library",
		}),
		artistsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_artists_total",
			Help:      "Total number of artists in library",
		}),
		albumsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_albums_total",
			Help:      "Total number of albums in library",
		}),
		genreCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_genre_count",
			Help:      "Number of tracks per genre",
		}, []string{"genre"}),
		formatCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_format_count",
			Help:      "Number of tracks per audio format",
		}, []string{"format"}),
		yearCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_year_count",
			Help:      "Number of tracks per release year",
		}, []string{"year"}),
		metadataComplete: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_metadata_complete",
			Help:      "Number of tracks with complete metadata (title, artist, album, genre, year)",
		}),
		metadataMissingGenre: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_metadata_missing_genre",
			Help:      "Number of tracks missing genre",
		}),
		metadataMissingYear: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_metadata_missing_year",
			Help:      "Number of tracks missing year",
		}),
		metadataMissingLyrics: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_metadata_missing_lyrics",
			Help:      "Number of tracks missing lyrics",
		}),
		lyricsPresent: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_lyrics_present",
			Help:      "Number of tracks with lyrics",
		}),
		lyricsMissing: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "library_lyrics_missing",
			Help:      "Number of tracks without lyrics",
		}),
		jobsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "jobs_total",
			Help:      "Number of jobs by status",
		}, []string{"status"}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "build_info",
			Help:      "Build information about the Soulsolid application",
		}, []string{"version", "commit", "branch"}),
	}
}

// Describe sends all metric descriptors to the channel.
func (c *PrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	c.tracksTotal.Describe(ch)
	c.artistsTotal.Describe(ch)
	c.albumsTotal.Describe(ch)
	c.genreCount.Describe(ch)
	c.formatCount.Describe(ch)
	c.yearCount.Describe(ch)
	c.metadataComplete.Describe(ch)
	c.metadataMissingGenre.Describe(ch)
	c.metadataMissingYear.Describe(ch)
	c.metadataMissingLyrics.Describe(ch)
	c.lyricsPresent.Describe(ch)
	c.lyricsMissing.Describe(ch)
	c.jobsTotal.Describe(ch)
	c.buildInfo.Describe(ch)
}

// Collect fetches metrics from the library and job service and sends them to the channel.
func (c *PrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	// Collect library metrics
	if totalTracks, err := c.libraryMetrics.GetTotalTracks(ctx); err != nil {
		slog.Warn("Failed to get track count for Prometheus", "error", err)
	} else {
		c.tracksTotal.Set(float64(totalTracks))
	}

	if totalArtists, err := c.libraryMetrics.GetTotalArtists(ctx); err != nil {
		slog.Warn("Failed to get artist count for Prometheus", "error", err)
	} else {
		c.artistsTotal.Set(float64(totalArtists))
	}

	if totalAlbums, err := c.libraryMetrics.GetTotalAlbums(ctx); err != nil {
		slog.Warn("Failed to get album count for Prometheus", "error", err)
	} else {
		c.albumsTotal.Set(float64(totalAlbums))
	}

	// Collect genre distribution
	if genreMetrics, err := c.libraryMetrics.GetStoredMetrics(ctx, "genre_counts"); err != nil {
		slog.Warn("Failed to get genre metrics for Prometheus", "error", err)
	} else {
		c.genreCount.Reset()
		for _, metric := range genreMetrics {
			c.genreCount.WithLabelValues(metric.Key).Set(float64(metric.Value))
		}
	}

	// Collect format distribution
	if formatMetrics, err := c.libraryMetrics.GetStoredMetrics(ctx, "format_distribution"); err != nil {
		slog.Warn("Failed to get format metrics for Prometheus", "error", err)
	} else {
		c.formatCount.Reset()
		for _, metric := range formatMetrics {
			c.formatCount.WithLabelValues(metric.Key).Set(float64(metric.Value))
		}
	}

	// Collect year distribution
	if yearMetrics, err := c.libraryMetrics.GetStoredMetrics(ctx, "year_distribution"); err != nil {
		slog.Warn("Failed to get year metrics for Prometheus", "error", err)
	} else {
		c.yearCount.Reset()
		for _, metric := range yearMetrics {
			c.yearCount.WithLabelValues(metric.Key).Set(float64(metric.Value))
		}
	}

	// Collect metadata completeness
	if metadataMetrics, err := c.libraryMetrics.GetStoredMetrics(ctx, "metadata_completeness"); err != nil {
		slog.Warn("Failed to get metadata completeness metrics for Prometheus", "error", err)
	} else {
		// Reset individual gauges
		c.metadataComplete.Set(0)
		c.metadataMissingGenre.Set(0)
		c.metadataMissingYear.Set(0)
		c.metadataMissingLyrics.Set(0)

		for _, metric := range metadataMetrics {
			switch metric.Key {
			case "complete":
				c.metadataComplete.Set(float64(metric.Value))
			case "missing_genre":
				c.metadataMissingGenre.Set(float64(metric.Value))
			case "missing_year":
				c.metadataMissingYear.Set(float64(metric.Value))
			case "missing_lyrics":
				c.metadataMissingLyrics.Set(float64(metric.Value))
			}
		}
	}

	// Collect lyrics stats
	if lyricsMetrics, err := c.libraryMetrics.GetStoredMetrics(ctx, "lyrics_stats"); err != nil {
		slog.Warn("Failed to get lyrics metrics for Prometheus", "error", err)
	} else {
		c.lyricsPresent.Set(0)
		c.lyricsMissing.Set(0)

		for _, metric := range lyricsMetrics {
			switch metric.Key {
			case "has_lyrics":
				c.lyricsPresent.Set(float64(metric.Value))
			case "no_lyrics":
				c.lyricsMissing.Set(float64(metric.Value))
			}
		}
	}

	// Collect job counts
	jobCounts := map[string]int{
		"pending":   0,
		"running":   0,
		"completed": 0,
		"failed":    0,
		"cancelled": 0,
	}

	for _, job := range c.jobService.GetJobs() {
		jobCounts[string(job.Status)]++
	}

	c.jobsTotal.Reset()
	for status, count := range jobCounts {
		c.jobsTotal.WithLabelValues(status).Set(float64(count))
	}

	// Set build info
	c.buildInfo.Reset()
	version := os.Getenv("IMAGE_TAG")
	if version == "" {
		version = "dev"
	}
	commit := os.Getenv("GIT_COMMIT")
	if commit == "" {
		commit = "unknown"
	}
	branch := os.Getenv("GIT_BRANCH")
	if branch == "" {
		branch = "unknown"
	}
	c.buildInfo.WithLabelValues(version, commit, branch).Set(1)

	// Send metrics
	c.tracksTotal.Collect(ch)
	c.artistsTotal.Collect(ch)
	c.albumsTotal.Collect(ch)
	c.genreCount.Collect(ch)
	c.formatCount.Collect(ch)
	c.yearCount.Collect(ch)
	c.metadataComplete.Collect(ch)
	c.metadataMissingGenre.Collect(ch)
	c.metadataMissingYear.Collect(ch)
	c.metadataMissingLyrics.Collect(ch)
	c.lyricsPresent.Collect(ch)
	c.lyricsMissing.Collect(ch)
	c.jobsTotal.Collect(ch)
	c.buildInfo.Collect(ch)
}
