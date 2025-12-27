package analyze

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the analyze feature
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// API routes for analysis operations
	analyze := app.Group("/analyze")
	analyze.Post("/acoustid", handler.StartAcoustIDAnalysis)
	analyze.Post("/lyrics", handler.StartLyricsAnalysis)
	analyze.Post("/reorganize", handler.StartReorganizeAnalysis)

	// UI routes for the analyze section
	ui := app.Group("/ui")
	ui.Get("/analyze", handler.RenderAnalyzeSection)
}
