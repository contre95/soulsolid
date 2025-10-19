package ui

import (
	"log/slog"
	"os"

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

func (h *Handler) getBaseRenderData() fiber.Map {
	version := os.Getenv("IMAGE_TAG")
	if version == "" {
		version = "dev"
	}
	cfg := h.configManager.Get()
	downloaders := make(map[string]interface{})
	for _, plugin := range cfg.Downloaders.Plugins {
		downloaders[plugin.Name] = struct{ Name, Icon string }{Name: plugin.Name, Icon: plugin.Icon}
	}
	return fiber.Map{
		"Version":     version,
		"Downloaders": downloaders,
		"SyncEnabled": cfg.Sync.Enabled,
		"Telegram":    cfg.Telegram,
	}
}

func (h *Handler) renderMain(c *fiber.Ctx, page string, data fiber.Map) error {
	baseData := h.getBaseRenderData()
	for k, v := range data {
		baseData[k] = v
	}

	if c.Get("HX-Request") != "true" {
		return c.Render("main", baseData)
	}

	return c.Render("sections/"+page, data)
}

// RenderAdmin renders the admin page.
func (h *Handler) RenderAdmin(c *fiber.Ctx) error {
	slog.Debug("RenderAdmin handler called")
	return h.renderMain(c, "dashboard", fiber.Map{
		"DashboardTrigger": ",revealed",
	})
}

// RenderDashboard renders the main dashboard page.
func (h *Handler) RenderDashboard(c *fiber.Ctx) error {
	slog.Debug("RenderDashboard handler called")
	return h.renderMain(c, "dashboard", fiber.Map{
		"Title":            "Dashboard",
		"DashboardTrigger": ",revealed",
		"SyncEnabled":      h.configManager.Get().Sync.Enabled,
	})
}

// RenderLibrarySection renders the library page.
func (h *Handler) RenderLibrarySection(c *fiber.Ctx) error {
	slog.Debug("RenderLibrary handler called")
	return h.renderMain(c, "library", fiber.Map{
		"Title":               "Library",
		"DefaultDownloadPath": h.configManager.Get().DownloadPath,
		"LibraryTrigger":      ",revealed",
	})
}

// RenderImportSection renders the organize page.
func (h *Handler) RenderImportSection(c *fiber.Ctx) error {
	slog.Debug("RenderImport handler called")
	return h.renderMain(c, "import", fiber.Map{
		"Title":         "Import",
		"ImportTrigger": ",revealed",
	})
}

// GetSettingsSection renders the settings form with current configuration values.
func (h *Handler) GetSettingsSection(c *fiber.Ctx) error {
	slog.Debug("GetSettings handler called")
	return h.renderMain(c, "settings", fiber.Map{
		"Title":           "Settings",
		"SettingsTrigger": ",revealed",
	})
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
	return h.renderMain(c, "sync", fiber.Map{
		"Title":       "Sync",
		"SyncTrigger": ",revealed",
		"SyncEnabled": h.configManager.Get().Sync.Enabled,
	})
}

// GetQuickActionsCard renders the quick actions card for the dashboard.
func (h *Handler) GetQuickActionsCard(c *fiber.Ctx) error {
	slog.Debug("GetQuickActionsCard handler called")
	return c.Render("cards/quick_actions", fiber.Map{})
}

// RenderJobsSection renders the jobs page.
func (h *Handler) RenderJobsSection(c *fiber.Ctx) error {
	slog.Debug("RenderJobsSection handler called")
	return h.renderMain(c, "jobs", fiber.Map{
		"Title":       "Jobs",
		"JobsTrigger": ",revealed",
	})
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
	return h.renderMain(c, "download", fiber.Map{
		"Title":             "Download",
		"DownloadTrigger":   ",revealed",
		"CurrentDownloader": downloader,
	})
}
