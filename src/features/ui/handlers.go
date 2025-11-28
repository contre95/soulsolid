package ui

import (
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the UI feature.
type Handler struct {
	configManager *config.Manager
}

// NewHandler creates a new handler for the UI feature.
func NewHandler(configManager *config.Manager) *Handler {
	return &Handler{
		configManager: configManager,
	}
}

// RenderDashboard renders the main dashboard page.
func (h *Handler) RenderDashboard(c *fiber.Ctx) error {
	slog.Debug("RenderDashboard handler called")
	data := fiber.Map{
		"Title":       "Dashboard",
		"SyncEnabled": h.configManager.Get().Sync.Enabled,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "dashboard"
		return c.Render("main", data)
	}
	return c.Render("sections/dashboard", data)
}

// GetQuickActionsCard renders the quick actions card for the dashboard.
func (h *Handler) GetQuickActionsCard(c *fiber.Ctx) error {
	slog.Debug("GetQuickActionsCard handler called")
	return c.Render("cards/quick_actions", fiber.Map{})
}
