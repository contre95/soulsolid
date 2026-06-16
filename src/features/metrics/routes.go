package metrics

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the metrics routes with the Fiber app.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	metrics := app.Group("/metrics")
	metrics.Get("/overview", handler.GetMetricsOverview)
	metrics.Get("/charts/genre", handler.GetGenreChartHTML)
	metrics.Get("/charts/year", handler.GetYearChartHTML)
	metrics.Get("/charts/format", handler.GetFormatChartHTML)
	metrics.Get("/charts/quality", handler.GetQualityChartHTML)
	metrics.Get("/charts/metadata", handler.GetMetadataChartHTML)
}
