package ui

import (
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/hosting/respond"
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
	return respond.Section(c, "dashboard", fiber.Map{"Title": "Dashboard"})
}

// GetQuickActionsCard renders the quick actions card for the dashboard.
func (h *Handler) GetQuickActionsCard(c *fiber.Ctx) error {
	slog.Debug("GetQuickActionsCard handler called")
	return respond.Partial(c, "cards/quick_actions", fiber.Map{})
}

// RenderAnalyzeSection renders the all analyze jobs page
func (h *Handler) RenderAnalyzeSection(c *fiber.Ctx) error {
	slog.Debug("Rendering all analyze jobs page")
	return respond.Section(c, "analyze", fiber.Map{"Title": "All Analyze Jobs"})
}
