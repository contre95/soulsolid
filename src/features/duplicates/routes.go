package duplicates

import (
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App, handler *Handler) {

	// API routes
	queueAPI := app.Group("/duplicates/queue")
	queueAPI.Post("/:id/:action", handler.ProcessQueueItem)
	queueAPI.Post("/group/fp/:groupKey/:action", handler.ProcessQueueGroup)
	queueAPI.Post("/clear", handler.ClearQueue)
	queueAPI.Get("/count", handler.QueueCount)

	// UI routes
	ui := app.Group("/ui")
	ui.Get("/duplicates/queue/header", handler.RenderQueueHeader)
	ui.Get("/duplicates/queue/items", handler.RenderQueueItems)
	ui.Get("/duplicates/queue/items/grouped", handler.RenderGroupedQueueItems)
	ui.Get("/duplicates/compare/:id", handler.RenderCompare)

	// Analyze routes
	analyze := app.Group("/analyze")
	analyze.Post("/duplicates", handler.StartDuplicatesAnalysis)

	analyzeUI := app.Group("/ui/analyze")
	analyzeUI.Get("/duplicates", handler.RenderAnalyzeDuplicatesSection)
}
