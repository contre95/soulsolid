package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/hosting/respond"
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

// RenderSettingsSection renders the settings form with current configuration values.
func (h *Handler) RenderSettingsSection(c *fiber.Ctx) error {
	slog.Debug("RenderSettings handler called")
	return respond.Section(c, "settings", fiber.Map{"Title": "Settings"})
}

// UpdateSettings handles the form submission to update configuration.
func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	slog.Info("Configuration update requested")

	// Get current config to preserve server settings
	currentConfig := h.configManager.Get()
	// Parse form data into a new config struct
	// TODO: We might want to add some validations probably, not sure if here.
	newConfig := &Config{
		// Not read from the form: libraryPath is immutable at runtime
		// (enforced by Manager.Update). Config file + restart only.
		LibraryPath:  currentConfig.LibraryPath,
		DownloadPath: c.FormValue("downloadPath"),
		Database:     currentConfig.Database, // Preserve database settings
		Import: Import{
			AutoStartWatcher: currentConfig.Import.AutoStartWatcher,
			Move:             c.FormValue("import.move") == "true",
			AlwaysQueue:      c.FormValue("import.always_queue") == "true",
			Duplicates:       c.FormValue("import.duplicates"),
			AllowMissingMetadata: AllowMissingMetadata{
				Artist: c.FormValue("import.allow_missing_metadata.artist") == "true",
				Album:  c.FormValue("import.allow_missing_metadata.album") == "true",
				Title:  c.FormValue("import.allow_missing_metadata.title") == "true",
				Year:   c.FormValue("import.allow_missing_metadata.year") == "true",
				Genre:  c.FormValue("import.allow_missing_metadata.genre") == "true",
			},
			PathOptions: Paths{
				DefaultPath:     c.FormValue("import.paths.default_path"),
				Compilations:    c.FormValue("import.paths.compilations"),
				AlbumSoundtrack: c.FormValue("import.paths.album:soundtrack"),
				AlbumSingle:     c.FormValue("import.paths.album:single"),
				AlbumEP:         c.FormValue("import.paths.album:ep"),
				Fat32Safe:       c.FormValue("import.paths.fat32_safe") == "true",
			},
		},
		Downloaders: Downloaders{
			Plugins: currentConfig.Downloaders.Plugins, // Preserve plugins
			Artwork: currentConfig.Downloaders.Artwork, // Preserve artwork settings
		},
		Metadata: Metadata{
			Providers: map[string]Provider{
				"musicbrainz": {
					Enabled: c.FormValue("metadata.providers.musicbrainz.enabled") == "true",
				},
				"discogs": {
					Enabled: c.FormValue("metadata.providers.discogs.enabled") == "true",
					Secret: func() *string {
						secret := c.FormValue("metadata.providers.discogs.secret")
						if secret != "" {
							return &secret
						}
						return nil
					}(),
				},
				"deezer": {
					Enabled: c.FormValue("metadata.providers.deezer.enabled") == "true",
				},
				"acoustid": {
					Enabled: c.FormValue("metadata.providers.acoustid.enabled") == "true",
					Secret: func() *string {
						secret := c.FormValue("metadata.providers.acoustid.secret")
						if secret != "" {
							return &secret
						}
						return nil
					}(),
				},
			},
		},
		Lyrics: currentConfig.Lyrics,
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
		Jobs: Jobs{
			Log:      c.FormValue("jobs.log") == "true",
			LogPath:  c.FormValue("jobs.log_path"),
			Webhooks: currentConfig.Jobs.Webhooks,
		},
	}

	return h.apply(c, newConfig, "Configuration updated successfully!")
}

// UpdateImportOptions saves the import options edited from the Import Directory card.
// Only these fields change; the rest of the config is kept as is.
func (h *Handler) UpdateImportOptions(c *fiber.Ctx) error {
	duplicates := c.FormValue("import.duplicates")
	switch duplicates {
	case "replace", "skip", "queue":
	default:
		return respond.ToastErr(c, fiber.StatusBadRequest, "Invalid duplicate handling: "+duplicates)
	}

	newConfig := *h.configManager.Get()
	newConfig.Import.Move = c.FormValue("import.move") == "true"
	newConfig.Import.AlwaysQueue = c.FormValue("import.always_queue") == "true"
	newConfig.Import.Duplicates = duplicates

	// Lets the card's badges re-render with the saved values.
	c.Set("HX-Trigger", "importOptionsChanged")
	return h.apply(c, &newConfig, "Import options saved")
}

// UpdatePathTemplates saves the path templates edited from the Path Templates card.
func (h *Handler) UpdatePathTemplates(c *fiber.Ctx) error {
	paths := h.configManager.Get().Import.PathOptions
	paths.DefaultPath = strings.TrimSpace(c.FormValue("import.paths.default_path"))
	paths.Compilations = strings.TrimSpace(c.FormValue("import.paths.compilations"))
	paths.AlbumSoundtrack = strings.TrimSpace(c.FormValue("import.paths.album:soundtrack"))
	paths.AlbumSingle = strings.TrimSpace(c.FormValue("import.paths.album:single"))
	paths.AlbumEP = strings.TrimSpace(c.FormValue("import.paths.album:ep"))

	// An empty template would render tracks straight into the library root.
	for _, t := range []struct{ name, tmpl string }{
		{"Default", paths.DefaultPath},
		{"Soundtrack", paths.AlbumSoundtrack},
		{"Single", paths.AlbumSingle},
		{"EP", paths.AlbumEP},
	} {
		if t.tmpl == "" {
			return respond.ToastErr(c, fiber.StatusBadRequest, t.name+" path template cannot be empty")
		}
	}

	newConfig := *h.configManager.Get()
	newConfig.Import.PathOptions = paths
	return h.apply(c, &newConfig, "Path templates saved")
}

// apply swaps in newConfig, persists it, and responds with a toast.
func (h *Handler) apply(c *fiber.Ctx, newConfig *Config, msg string) error {
	h.configManager.Update(newConfig)
	slog.Info("Configuration updated in memory")
	if err := h.configManager.Save(); err != nil {
		slog.Warn("failed to save config to file (this is normal in containerized environments)", "error", err)
	} else {
		slog.Info("Configuration saved to file successfully")
	}
	return respond.ToastOk(c, msg)
}

func (h *Handler) GetConfigForm(c *fiber.Ctx) error {
	slog.Debug("GetSettingsForm handler called")
	config := h.configManager.Get()

	return respond.Partial(c, "config/config_form", fiber.Map{
		"Config": config,
	})
}

// GetConfig returns the config as JSON, or as raw YAML text when ?fmt=yaml is set.
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	slog.Debug("GetConfig handler called")
	if c.Query("fmt") == "yaml" {
		c.Set("Content-Type", "text/yaml")
		return c.SendString(h.configManager.GetYAML())
	}
	return c.JSON(h.configManager.Get())
}

// DownloadDatabase serves the database file for download.
func (h *Handler) DownloadDatabase(c *fiber.Ctx) error {
	slog.Debug("DownloadDatabase handler called")

	config := h.configManager.Get()
	dbPath := config.Database.Path

	if dbPath == "" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "Database path not configured")
	}

	filename := filepath.Base(dbPath)
	return respond.Resource(c, "application/octet-stream", fmt.Sprintf("%s/config/database/download", c.BaseURL()), func() error {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Set("Content-Type", "application/octet-stream")
		return c.SendFile(dbPath)
	})
}
