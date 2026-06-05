package streaming

import "github.com/gofiber/fiber/v2"

// RegisterRoutes registers the audio streaming routes.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)
	app.Get("/stream/queue/:id", handler.StreamQueueItem)
	app.Get("/stream/library/:id", handler.StreamLibraryTrack)
}
