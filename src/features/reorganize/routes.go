package reorganize

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the reorganize feature.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// API routes for file reorganization
	app.Post("/analyze/reorganize", handler.StartReorganizeAnalysis)

	// UI routes for the file reorganization section
	ui := app.Group("/ui")
	ui.Get("/analyze/files", handler.RenderFilesReorganizationSection)
}
