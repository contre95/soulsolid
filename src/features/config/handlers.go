package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the config feature.
type Handler struct {
	configManager *Manager
}

// NewHandler creates a new handler for the config feature.
func NewHandler(configManager *Manager) *Handler {
	return &Handler{
		configManager: configManager,
	}
}

// UpdateSettings handles the form submission to update configuration.
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	slog.Info("Configuration update requested")

	// Get current config to preserve server settings
	currentConfig := h.configManager.Get()
	slog.Debug(h.configManager.GetJSON())
	// Parse form data into a new config struct
	// TODO: We might want to add some validations probably, not sure if here.
	slog.Warn("Method not fully implemented")
	newConfig := &Config{
		LibraryPath:  c.FormValue("libraryPath"),
		DownloadPath: c.FormValue("downloadPath"),
		Import: Import{
			Move:        c.FormValue("import.move") == "true",
			AlwaysQueue: c.FormValue("import.always_queue") == "true",
			Duplicates:  c.FormValue("import.duplicates"),
			PathOptions: Paths{
				DefaultPath:     c.FormValue("import.paths.default_path"),
				Compilations:    c.FormValue("import.paths.compilations"),
				AlbumSoundtrack: c.FormValue("import.paths.album:soundtrack"),
				AlbumSingle:     c.FormValue("import.paths.album:single"),
				AlbumEP:         c.FormValue("import.paths.album:ep"),
			},
		},
		Telegram: Telegram{
			Enabled:      c.FormValue("telegram.enabled") == "true",
			Token:        c.FormValue("telegram.token"),
			AllowedUsers: parseStringSlice(c.FormValue("telegram.allowedUsers")),
			BotHandle:    c.FormValue("telegram.bot_handle"),
		},
		Downloaders: Downloaders{
			Plugins: currentConfig.Downloaders.Plugins, // Preserve plugins
			Artwork: currentConfig.Downloaders.Artwork, // Preserve artwork settings
			TagFile: currentConfig.Downloaders.TagFile, // Preserve tag_file
		},
		Metadata: Metadata{
			Providers: map[string]Provider{
				"musicbrainz": {
					Enabled: c.FormValue("metadata.providers.musicbrainz.enabled") == "true",
				},
				"discogs": {
					Enabled: c.FormValue("metadata.providers.discogs.enabled") == "true",
					APIKey:  c.FormValue("metadata.providers.discogs.api_key"),
				},
				"deezer": {
					Enabled: c.FormValue("metadata.providers.deezer.enabled") == "true",
				},
			},
		},
		// Preserve server settings from current config, no sense to be changed on runtime
		Server: Server{
			Port:        currentConfig.Server.Port,
			PrintRoutes: currentConfig.Server.PrintRoutes,
		},
		Logger: Logger{
			Enabled:   c.FormValue("logger.enabled") == "true",
			Level:     c.FormValue("logger.level"),
			Format:    c.FormValue("logger.format"),
			HTMXDebug: c.FormValue("logger.htmx_debug") == "true",
		},
		Sync: Sync{
			Enabled: currentConfig.Sync.Enabled,
			Devices: currentConfig.Sync.Devices,
		},
		Jobs: Jobs{
			Log:      c.FormValue("jobs.log") == "true",
			LogPath:  c.FormValue("jobs.log_path"),
			Webhooks: currentConfig.Jobs.Webhooks,
		},
	}

	// Update the configuration
	h.configManager.Update(newConfig)
	slog.Info("Configuration updated in memory")

	// Try to save to file (optional - may fail in containerized environments)
	if err := h.configManager.Save("config.yaml"); err != nil {
		slog.Warn("failed to save config to file (this is normal in containerized environments)", "error", err)
	} else {
		slog.Info("Configuration saved to file successfully")
	}

	return c.Render("toast/toastOk", fiber.Map{
		"Msg": "Configuration updated successfully!",
	})
}

// Helper functions for parsing form values
func parseInt(s string) int {
	var result int
	if s != "" {
		_, err := fmt.Sscanf(s, "%d", &result)
		if err != nil {
			return 0
		}
	}
	return result
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	// Split by comma and trim spaces
	var result []string
	for part := range strings.SplitSeq(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (h *Handler) GetConfigForm(c *fiber.Ctx) error {
	slog.Debug("GetSettingsForm handler called")
	config := h.configManager.Get()

	return c.Render("config/config_form", fiber.Map{
		"Config": config,
	})
}

// GetConfig returns the current configuration in the requested format.
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	slog.Debug("GetConfig handler called", "format", c.Query("fmt", "json"))
	format := c.Query("fmt", "yaml")

	switch format {
	case "yaml":
		c.Set("Content-Type", "text/yaml")
		return c.SendString(h.configManager.GetYAML())
	case "json":
		c.Set("Content-Type", "application/json")
		return c.SendString(h.configManager.GetJSON())
	default:
		return c.Status(fiber.StatusBadRequest).SendString("Invalid format. Use 'json' or 'yaml'")
	}
}

// DownloadDatabase serves the database file for download.
func (h *Handler) DownloadDatabase(c *fiber.Ctx) error {
	slog.Debug("DownloadDatabase handler called")

	config := h.configManager.Get()
	dbPath := config.Database.Path

	if dbPath == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Database path not configured")
	}

	// Extract filename from path for download
	filename := filepath.Base(dbPath)

	// Set headers for file download
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set("Content-Type", "application/octet-stream")

	// Send the file
	return c.SendFile(dbPath)
}
