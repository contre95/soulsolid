package syncdap

import (
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers sync-related routes
func RegisterRoutes(app *fiber.App, service *Service, jobService jobs.JobService) {
	handler := NewHandler(service, jobService)

	// UI routes - use existing /ui group
	ui := app.Group("/ui")
	ui.Get("/sync/status", handler.GetSyncStatus)
	ui.Get("/sync/device-status-card", handler.GetDeviceStatusCard)
	// Sync API routes
	app.Get("/sync/device/:uuid", handler.GetDeviceStatus)
	app.Post("/sync/device/:uuid/trigger", handler.TriggerSync)
	app.Post("/sync/device/:uuid/cancel", handler.CancelSync)
}
