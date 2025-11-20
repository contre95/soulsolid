package syncdap

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers sync-related routes
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// UI routes - use existing /ui group
	ui := app.Group("/ui")
	ui.Get("/sync/status", handler.GetSyncStatus)

	// Sync API routes
	app.Get("/sync/device/:uuid", handler.GetDeviceStatus)
	app.Post("/sync/device/:uuid/trigger", handler.TriggerSync)
	app.Post("/sync/device/:uuid/cancel", handler.CancelSync)
}
