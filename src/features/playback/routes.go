package playback

import "github.com/gofiber/fiber/v2"

// RegisterRoutes registers the playback routes
func RegisterRoutes(app *fiber.App, handler *Handler) {
	playback := app.Group("/playback")
	playback.Get("/tracks/:id/preview", handler.GetTrackPreview)
}
