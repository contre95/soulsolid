package metrics

import (
	"log/slog"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for the metrics feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new metrics handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetMetricsOverview renders the metrics overview page.
func (h *Handler) GetMetricsOverview(c *fiber.Ctx) error {
	slog.Debug("GetMetricsOverview handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading metrics")
	}

	// Check if request accepts HTML (HTMX request)
	acceptHeader := c.Get("Accept")
	hxRequest := c.Get("HX-Request")
	if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
		return c.Render("metrics/overview", fiber.Map{
			"Metrics": metrics,
		})
	}

	return c.JSON(metrics)
}

// GetGenreChart returns genre distribution data for charts.
func (h *Handler) GetGenreChart(c *fiber.Ctx) error {
	slog.Debug("GetGenreChart handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load metrics",
		})
	}

	chartData := metrics.GenreChartData()
	return c.JSON(chartData)
}

// GetLyricsChart returns lyrics statistics data for charts.
func (h *Handler) GetLyricsChart(c *fiber.Ctx) error {
	slog.Debug("GetLyricsChart handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load metrics",
		})
	}

	chartData := metrics.LyricsChartData()
	return c.JSON(chartData)
}

// GetMetadataChart returns metadata completeness data for charts.
func (h *Handler) GetMetadataChart(c *fiber.Ctx) error {
	slog.Debug("GetMetadataChart handler called")

	totalTracks, err := h.service.metrics.GetTotalTracks(c.Context())
	if err != nil {
		slog.Error("Error getting total tracks", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load metrics",
		})
	}

	if totalTracks == 0 {
		return c.JSON(&ChartData{
			Labels:   []string{},
			Datasets: []Dataset{},
		})
	}

	// Get counts for each metadata field
	isrcCount, err := h.service.metrics.GetTracksWithISRC(c.Context())
	if err != nil {
		slog.Error("Error getting ISRC count", "error", err)
		isrcCount = 0
	}
	bpmCount, err := h.service.metrics.GetTracksWithValidBPM(c.Context())
	if err != nil {
		slog.Error("Error getting BPM count", "error", err)
		bpmCount = 0
	}
	yearCount, err := h.service.metrics.GetTracksWithValidYear(c.Context())
	if err != nil {
		slog.Error("Error getting year count", "error", err)
		yearCount = 0
	}
	genreCount, err := h.service.metrics.GetTracksWithValidGenre(c.Context())
	if err != nil {
		slog.Error("Error getting genre count", "error", err)
		genreCount = 0
	}
	lyricsStats, err := h.service.metrics.GetLyricsStats(c.Context())
	if err != nil {
		slog.Error("Error getting lyrics stats", "error", err)
		lyricsStats = music.LyricsStats{}
	}

	// Calculate percentages
	isrcPct := float64(isrcCount) / float64(totalTracks) * 100
	bpmPct := float64(bpmCount) / float64(totalTracks) * 100
	yearPct := float64(yearCount) / float64(totalTracks) * 100
	genrePct := float64(genreCount) / float64(totalTracks) * 100
	lyricsPct := float64(lyricsStats.WithLyrics) / float64(totalTracks) * 100

	labels := []string{"ISRC", "BPM", "Year", "Genre", "Lyrics"}
	data := []float64{isrcPct, bpmPct, yearPct, genrePct, lyricsPct}

	return c.JSON(&ChartData{
		Labels: labels,
		Datasets: []Dataset{{
			Label:           "Metadata Completeness (%)",
			Data:            data,
			BackgroundColor: []string{"#4BC0C0", "#FFCE56", "#FF6384", "#36A2EB", "#9966FF"},
		}},
	})
}

// GetYearChart returns year distribution data for charts.
func (h *Handler) GetYearChart(c *fiber.Ctx) error {
	slog.Debug("GetYearChart handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load metrics",
		})
	}

	chartData := metrics.YearBarData()
	return c.JSON(chartData)
}

// GetFormatChart returns format distribution data for charts.
func (h *Handler) GetFormatChart(c *fiber.Ctx) error {
	slog.Debug("GetFormatChart handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load metrics",
		})
	}

	chartData := metrics.FormatBarData()
	return c.JSON(chartData)
}
