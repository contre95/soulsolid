package config

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the config feature.
func RegisterRoutes(app *fiber.App, configManager *Manager) {
	// Create a new handler for the config feature.
	handler := NewHandler(configManager)

	/// UI
	ui := app.Group("/ui")
	ui.Get("/config/form", handler.GetConfigForm)

	// APP
	app.Post("/settings/update", handler.UpdateSettings)
	app.Get("/config", handler.GetConfig)
	app.Get("/config/database/download", handler.DownloadDatabase)
}
