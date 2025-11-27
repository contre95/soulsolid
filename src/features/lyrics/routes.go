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
	tagGroup.Get("/edit/:trackId/lyrics/text/:provider", handler.GetLyricsText)
	tagGroup.Get("/buttons/lyrics/:trackId", handler.RenderLyricsButtons)

	// Library routes for lyrics
	library := app.Group("/library")
	library.Get("/tracks/:id/lyrics", handler.GetTrackLyrics)
}
