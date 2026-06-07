package ui

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the UI feature.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	app.Get("/", handler.RenderDashboard)
	app.Get("/dashboard", handler.RenderDashboard)
	app.Get("/analyze", handler.RenderAnalyzeSection)
	app.Get("/dashboard/quick-actions", handler.GetQuickActionsCard)
}
