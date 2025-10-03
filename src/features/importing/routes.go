package importing

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the importing feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	ui := app.Group("/ui")
	// UI endpoints
	ui.Get("/importing/directory/form", handler.GetDirectoryForm)
	ui.Get("/importing/queue/items", handler.RenderQueueItems)
	ui.Get("/importing/queue/header", handler.GetQueueHeader)

	// Action endpoints
	app.Post("/import/directory", handler.ImportDirectory)
	app.Post("/import/queue/:id/:action", handler.ProcessQueueItem)
	app.Post("/import/queue/clear", handler.ClearQueue)
	app.Get("/import/queue/count", handler.QueueCount)

}
