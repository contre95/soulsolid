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
	ui.Get("/", handler.RenderDashboard)
	ui.Get("/dashboard", handler.RenderDashboard)

	// Dashboard card endpoints
	ui.Get("/quick-actions-card", handler.GetQuickActionsCard)

}
