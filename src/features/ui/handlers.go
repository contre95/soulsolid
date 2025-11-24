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

// RenderAdmin renders the admin page.
func (h *Handler) RenderAdmin(c *fiber.Ctx) error {
	slog.Debug("RenderAdmin handler called")
	if c.Get("HX-Request") != "true" {
		return c.Render("main", fiber.Map{
			"Section": "dashboard",
		})
	}
	return c.Render("sections/dashboard", fiber.Map{})
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

// RenderLibrarySection renders the library page.
func (h *Handler) RenderLibrarySection(c *fiber.Ctx) error {
	slog.Debug("RenderLibrary handler called")
	data := fiber.Map{
		"Title":               "Library",
		"DefaultDownloadPath": h.configManager.Get().DownloadPath,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "library"
		return c.Render("main", data)
	}
	return c.Render("sections/library", data)
}

// RenderImportSection renders the organize page.
func (h *Handler) RenderImportSection(c *fiber.Ctx) error {
	slog.Debug("RenderImport handler called")
	data := fiber.Map{
		"Title": "Import",
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "import"
		return c.Render("main", data)
	}
	return c.Render("sections/import", data)
}

// GetSettingsSection renders the settings form with current configuration values.
func (h *Handler) GetSettingsSection(c *fiber.Ctx) error {
	slog.Debug("GetSettings handler called")
	data := fiber.Map{
		"Title": "Settings",
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "settings"
		return c.Render("main", data)
	}
	return c.Render("sections/settings", data)
}

// RenderSyncStatus renders the sync status page.
func (h *Handler) RenderSyncStatus(c *fiber.Ctx) error {
	slog.Debug("RenderSyncStatus handler called")
	return c.Render("sync/sync_status", fiber.Map{
		"Title": "Sync Status",
	})
}

// RenderSyncPage renders the sync page.
func (h *Handler) RenderSyncPage(c *fiber.Ctx) error {
	slog.Debug("RenderSyncPage handler called")
	data := fiber.Map{
		"Title":       "Sync",
		"SyncEnabled": h.configManager.Get().Sync.Enabled,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "sync"
		return c.Render("main", data)
	}
	return c.Render("sections/sync", data)
}

// GetQuickActionsCard renders the quick actions card for the dashboard.
func (h *Handler) GetQuickActionsCard(c *fiber.Ctx) error {
	slog.Debug("GetQuickActionsCard handler called")
	return c.Render("cards/quick_actions", fiber.Map{})
}

// RenderJobsSection renders the jobs page.
func (h *Handler) RenderJobsSection(c *fiber.Ctx) error {
	slog.Debug("RenderJobsSection handler called")
	data := fiber.Map{
		"Title": "Jobs",
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "jobs"
		return c.Render("main", data)
	}
	return c.Render("sections/jobs", data)
}

// RenderDownloadSection renders the download page.
func (h *Handler) RenderDownloadSection(c *fiber.Ctx) error {
	slog.Debug("RenderDownloadSection handler called")
	downloader := c.Query("downloader", "")
	if downloader == "" {
		cfg := h.configManager.Get()
		if len(cfg.Downloaders.Plugins) > 0 {
			downloader = cfg.Downloaders.Plugins[0].Name
		}
	}
	data := fiber.Map{
		"Title":             "Download",
		"CurrentDownloader": downloader,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "download"
		return c.Render("main", data)
	}
	return c.Render("sections/download", data)
}

// RenderAnalyzeSection renders the analyze page.
func (h *Handler) RenderAnalyzeSection(c *fiber.Ctx) error {
	slog.Debug("RenderAnalyzeSection handler called")
	data := fiber.Map{
		"Title": "Analyze",
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "analyze"
		return c.Render("main", data)
	}
	return c.Render("sections/analyze", data)
}
