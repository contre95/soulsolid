package importing

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the importing feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	app.Get("/import", handler.RenderImportSection)

	importGroup := app.Group("/import")
	importGroup.Get("/directory/form", handler.GetDirectoryForm)
	importGroup.Get("/queue/items", handler.RenderQueueItems)
	importGroup.Get("/queue/items/grouped", handler.RenderGroupedQueueItems)
	importGroup.Get("/queue/header", handler.GetQueueHeader)
	importGroup.Get("/queue/:id/artwork", handler.ServeQueueItemArtwork)
	importGroup.Post("/directory", handler.ImportDirectory)
	importGroup.Post("/queue/:id/:action", handler.ProcessQueueItem)
	importGroup.Post("/queue/group/:groupType/:groupKey/:action", handler.ProcessQueueGroup)
	importGroup.Post("/queue/clear", handler.ClearQueue)
	importGroup.Post("/prune/download-path", handler.PruneDownloadPath)
	importGroup.Get("/queue/count", handler.QueueCount)
	importGroup.Post("/watcher/toggle", handler.ToggleWatcher)
	importGroup.Get("/watcher/status", handler.GetWatcherStatus)
	importGroup.Get("/watcher/toggle-state", handler.GetWatcherToggleState)
}
