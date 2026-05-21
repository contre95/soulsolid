package reorganize

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the reorganize feature.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	app.Post("/analyze/reorganize", handler.StartReorganizeAnalysis)
	app.Get("/analyze/files", handler.RenderFilesReorganizationSection)
}
