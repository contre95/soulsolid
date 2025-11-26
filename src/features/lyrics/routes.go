package lyrics

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers lyrics routes
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// UI routes for HTMX partials
	ui := app.Group("/ui")
	tagGroup := ui.Group("/tag")

	// Lyrics routes - these are accessed from the metadata/tag UI
	tagGroup.Get("/edit/:trackId/lyrics/best", handler.FetchBestLyrics)
	tagGroup.Get("/edit/:trackId/lyrics/:provider", handler.FetchLyricsFromProvider)
	tagGroup.Post("/edit/:trackId/lyrics/select/:provider", handler.SelectLyricsFromProvider)

	// API routes
	api := app.Group("/api")
	lyricsAPI := api.Group("/lyrics")

	// API endpoints for lyrics operations
	lyricsAPI.Get("/search/:trackId/:provider", handler.FetchLyricsFromProvider)
	lyricsAPI.Post("/select/:trackId/:provider", handler.SelectLyricsFromProvider)
}
