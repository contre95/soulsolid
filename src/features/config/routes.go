package config

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the config feature.
func RegisterRoutes(app *fiber.App, configManager *Manager) {
	handler := NewHandler(configManager)

	app.Get("/settings", handler.RenderSettingsSection)
	app.Get("/config/form", handler.GetConfigForm)
	app.Put("/settings", handler.UpdateSettings)
	app.Get("/config", handler.GetConfig)
	app.Get("/config/database/download", handler.DownloadDatabase)
}
