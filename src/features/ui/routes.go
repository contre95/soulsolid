package ui

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the UI feature.
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// Create a new group for the UI feature.
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/ui")
	})
	ui := app.Group("/ui")
	// Register the routes for pages.
	ui.Get("/", handler.RenderAdmin)
	ui.Get("/library", handler.RenderLibrarySection)
	ui.Get("/import", handler.RenderImportSection)
	ui.Get("/settings", handler.GetSettingsSection)
	ui.Get("/jobs", handler.RenderJobsSection)
	ui.Get("/download", handler.RenderDownloadSection)
	ui.Get("/dashboard", handler.RenderDashboard)
	ui.Get("/sync-status", handler.RenderSyncStatus)
	ui.Get("/sync", handler.RenderSyncPage)

	// Dashboard card endpoints

	ui.Get("/quick-actions-card", handler.GetQuickActionsCard)

}
