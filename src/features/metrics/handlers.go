package metrics

import (
	"log/slog"
	"strings"

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

// GetGenreChartHTML returns genre chart as HTML fragment for HTMX.
func (h *Handler) GetGenreChartHTML(c *fiber.Ctx) error {
	slog.Debug("GetGenreChartHTML handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading chart data")
	}

	chartData := metrics.GenreChartData()
	return c.Render("metrics/charts/genre_treemap", fiber.Map{
		"ChartData": chartData,
	})
}

// GetYearChartHTML returns year chart as HTML fragment for HTMX.
func (h *Handler) GetYearChartHTML(c *fiber.Ctx) error {
	slog.Debug("GetYearChartHTML handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading chart data")
	}

	chartData := metrics.YearBarData()
	return c.Render("metrics/charts/year_vbars", fiber.Map{
		"ChartData": chartData,
	})
}

// GetFormatChartHTML returns format chart as HTML fragment for HTMX.
func (h *Handler) GetFormatChartHTML(c *fiber.Ctx) error {
	slog.Debug("GetFormatChartHTML handler called")

	metrics, err := h.service.GetAllMetrics(c.Context())
	if err != nil {
		slog.Error("Error loading metrics for chart", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading chart data")
	}

	chartData := metrics.FormatBarData()
	return c.Render("metrics/charts/format_pie", fiber.Map{
		"ChartData": chartData,
	})
}

// GetMetadataChartHTML returns metadata chart as HTML fragment for HTMX.
func (h *Handler) GetMetadataChartHTML(c *fiber.Ctx) error {
	slog.Debug("GetMetadataChartHTML handler called")

	totalTracks, err := h.service.metrics.GetTotalTracks(c.Context())
	if err != nil {
		slog.Error("Error getting total tracks", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading chart data")
	}

	if totalTracks == 0 {
		return c.Render("metrics/charts/metadata_hbars", fiber.Map{
			"ChartData": nil,
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
		lyricsStats = LyricsStats{}
	}

	// Calculate percentages
	isrcPct := float64(isrcCount) / float64(totalTracks) * 100
	bpmPct := float64(bpmCount) / float64(totalTracks) * 100
	yearPct := float64(yearCount) / float64(totalTracks) * 100
	genrePct := float64(genreCount) / float64(totalTracks) * 100
	lyricsPct := float64(lyricsStats.WithLyrics) / float64(totalTracks) * 100

	labels := []string{"ISRC", "BPM", "Year", "Genre", "Lyrics"}
	data := []float64{isrcPct, bpmPct, yearPct, genrePct, lyricsPct}

	chartData := &ApexChartData{
		Labels: labels,
		Series: data,
		Colors: []string{"#00E396", "#FEB019", "#FF4560", "#008FFB", "#775DD0"},
	}

	return c.Render("metrics/charts/metadata_hbars", fiber.Map{
		"ChartData": chartData,
	})
}
