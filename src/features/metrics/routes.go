package metrics

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the metrics routes with the Fiber app.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// API routes for chart data
	api := app.Group("/api/metrics")
	api.Get("/genre-chart", handler.GetGenreChart)
	api.Get("/lyrics-chart", handler.GetLyricsChart)
	api.Get("/metadata-chart", handler.GetMetadataChart)
	api.Get("/year-chart", handler.GetYearChart)
	api.Get("/format-chart", handler.GetFormatChart)

	// UI routes for HTMX partials
	ui := app.Group("/ui/metrics")
	ui.Get("/overview", handler.GetMetricsOverview)

	// HTMX chart endpoints
	ui.Get("/charts/genre", handler.GetGenreChartHTML)
	ui.Get("/charts/year", handler.GetYearChartHTML)
	ui.Get("/charts/format", handler.GetFormatChartHTML)
	ui.Get("/charts/metadata", handler.GetMetadataChartHTML)
}
