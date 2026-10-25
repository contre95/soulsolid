package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/hosting/respond"
	"github.com/gofiber/fiber/v3"
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
func (h *Handler) RenderSettingsSection(c fiber.Ctx) error {
	slog.Debug("RenderSettings handler called")
	return respond.Section(c, "settings", fiber.Map{"Title": "Settings"})
}

// importOptionsForm holds the import options shared by the settings form and the Import Directory card.
type importOptionsForm struct {
	Move        bool   `form:"move"`
	AlwaysQueue bool   `form:"always_queue"`
	Duplicates  string `form:"duplicates" validate:"oneof=replace skip queue"`
}

// pathTemplatesForm holds the path templates shared by the settings form and the Path Templates card.
type pathTemplatesForm struct {
	DefaultPath     string `form:"default_path"`
	Compilations    string `form:"compilations"`
	AlbumSoundtrack string `form:"album:soundtrack"`
	AlbumSingle     string `form:"album:single"`
	AlbumEP         string `form:"album:ep"`
	Fat32Safe       bool   `form:"fat32_safe"`
}

type providerForm struct {
	Enabled bool   `form:"enabled"`
	Secret  string `form:"secret"`
}

// secret returns nil for an empty secret, so it's left out of the saved config.
func (p providerForm) secret() *string {
	if p.Secret == "" {
		return nil
	}
	return &p.Secret
}

// settingsForm mirrors the field names of views/config/config_form.html.
// Dotted names like "import.paths.fat32_safe" bind to the nested structs.
type settingsForm struct {
	DownloadPath string `form:"downloadPath"`
	Import       struct {
		importOptionsForm
		AllowMissingMetadata struct {
			Artist bool `form:"artist"`
			Album  bool `form:"album"`
			Title  bool `form:"title"`
			Year   bool `form:"year"`
			Genre  bool `form:"genre"`
		} `form:"allow_missing_metadata"`
		Paths pathTemplatesForm `form:"paths"`
	} `form:"import"`
	Telegram struct {
		Enabled      bool   `form:"enabled"`
		Token        string `form:"token"`
		AllowedUsers string `form:"allowedUsers"`
		BotHandle    string `form:"bot_handle"`
	} `form:"telegram"`
	Metadata struct {
		Providers struct {
			MusicBrainz providerForm `form:"musicbrainz"`
			Discogs     providerForm `form:"discogs"`
			Deezer      providerForm `form:"deezer"`
			AcoustID    providerForm `form:"acoustid"`
		} `form:"providers"`
	} `form:"metadata"`
	Logger struct {
		Enabled   bool   `form:"enabled"`
		Level     string `form:"level"`
		Format    string `form:"format"`
		HTMXDebug bool   `form:"htmx_debug"`
	} `form:"logger"`
	Jobs struct {
		Log     bool   `form:"log"`
		LogPath string `form:"log_path"`
	} `form:"jobs"`
}

// UpdateSettings handles the form submission to update configuration.
func (h *Handler) UpdateSettings(c fiber.Ctx) error {
	slog.Info("Configuration update requested")

	var form settingsForm
	if err := c.Bind().Form(&form); err != nil {
		return respond.BindErr(c, err)
	}

	// Get current config to preserve server settings
	currentConfig := h.configManager.Get()
	providers := form.Metadata.Providers
	paths := form.Import.Paths
	newConfig := &Config{
		// Not read from the form: libraryPath is immutable at runtime
		// (enforced by Manager.Update). Config file + restart only.
		LibraryPath:  currentConfig.LibraryPath,
		DownloadPath: form.DownloadPath,
		Database:     currentConfig.Database, // Preserve database settings
		Import: Import{
			AutoStartWatcher:     currentConfig.Import.AutoStartWatcher,
			Move:                 form.Import.Move,
			AlwaysQueue:          form.Import.AlwaysQueue,
			Duplicates:           form.Import.Duplicates,
			AllowMissingMetadata: AllowMissingMetadata(form.Import.AllowMissingMetadata),
			PathOptions: Paths{
				DefaultPath:     paths.DefaultPath,
				Compilations:    paths.Compilations,
				AlbumSoundtrack: paths.AlbumSoundtrack,
				AlbumSingle:     paths.AlbumSingle,
				AlbumEP:         paths.AlbumEP,
				Fat32Safe:       paths.Fat32Safe,
			},
		},
		Telegram: Telegram{
			Enabled:      form.Telegram.Enabled,
			Token:        form.Telegram.Token,
			AllowedUsers: parseStringSlice(form.Telegram.AllowedUsers),
			BotHandle:    form.Telegram.BotHandle,
		},
		Downloaders: Downloaders{
			Plugins: currentConfig.Downloaders.Plugins, // Preserve plugins
			Artwork: currentConfig.Downloaders.Artwork, // Preserve artwork settings
		},
		Metadata: Metadata{
			Providers: map[string]Provider{
				"musicbrainz": {Enabled: providers.MusicBrainz.Enabled},
				"discogs":     {Enabled: providers.Discogs.Enabled, Secret: providers.Discogs.secret()},
				"deezer":      {Enabled: providers.Deezer.Enabled},
				"acoustid":    {Enabled: providers.AcoustID.Enabled, Secret: providers.AcoustID.secret()},
			},
		},
		Lyrics: currentConfig.Lyrics,
		// Preserve server settings from current config, no sense to be changed on runtime
		Server: Server{
			Port:        currentConfig.Server.Port,
			PrintRoutes: currentConfig.Server.PrintRoutes,
		},
		Logger: Logger(form.Logger),
		Jobs: Jobs{
			Log:      form.Jobs.Log,
			LogPath:  form.Jobs.LogPath,
			Webhooks: currentConfig.Jobs.Webhooks,
		},
	}

	return h.apply(c, newConfig, "Configuration updated successfully!")
}

// UpdateImportOptions saves the import options edited from the Import Directory card.
// Only these fields change; the rest of the config is kept as is.
func (h *Handler) UpdateImportOptions(c fiber.Ctx) error {
	var form struct {
		Import importOptionsForm `form:"import"`
	}
	if err := c.Bind().Form(&form); err != nil {
		return respond.BindErr(c, err)
	}

	newConfig := *h.configManager.Get()
	newConfig.Import.Move = form.Import.Move
	newConfig.Import.AlwaysQueue = form.Import.AlwaysQueue
	newConfig.Import.Duplicates = form.Import.Duplicates

	// Lets the card's badges re-render with the saved values.
	respond.Trigger(c, "importOptionsChanged")
	return h.apply(c, &newConfig, "Import options saved")
}

// UpdatePathTemplates saves the path templates edited from the Path Templates card.
func (h *Handler) UpdatePathTemplates(c fiber.Ctx) error {
	var form struct {
		Import struct {
			Paths pathTemplatesForm `form:"paths"`
		} `form:"import"`
	}
	if err := c.Bind().Form(&form); err != nil {
		return respond.BindErr(c, err)
	}

	paths := h.configManager.Get().Import.PathOptions
	paths.DefaultPath = strings.TrimSpace(form.Import.Paths.DefaultPath)
	paths.Compilations = strings.TrimSpace(form.Import.Paths.Compilations)
	paths.AlbumSoundtrack = strings.TrimSpace(form.Import.Paths.AlbumSoundtrack)
	paths.AlbumSingle = strings.TrimSpace(form.Import.Paths.AlbumSingle)
	paths.AlbumEP = strings.TrimSpace(form.Import.Paths.AlbumEP)

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
func (h *Handler) apply(c fiber.Ctx, newConfig *Config, msg string) error {
	h.configManager.Update(newConfig)
	slog.Info("Configuration updated in memory")
	if err := h.configManager.Save(); err != nil {
		slog.Warn("failed to save config to file (this is normal in containerized environments)", "error", err)
	} else {
		slog.Info("Configuration saved to file successfully")
	}
	return respond.ToastOk(c, msg)
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

func (h *Handler) GetConfigForm(c fiber.Ctx) error {
	slog.Debug("GetSettingsForm handler called")
	config := h.configManager.Get()

	return respond.Partial(c, "config/config_form", fiber.Map{
		"Config": config,
	})
}

// GetConfig returns the config as JSON, or as raw YAML text when ?fmt=yaml is set.
func (h *Handler) GetConfig(c fiber.Ctx) error {
	slog.Debug("GetConfig handler called")
	if c.Query("fmt") == "yaml" {
		c.Set("Content-Type", "text/yaml")
		return c.SendString(h.configManager.GetYAML())
	}
	return c.JSON(h.configManager.Get())
}

// DownloadDatabase serves the database file for download.
func (h *Handler) DownloadDatabase(c fiber.Ctx) error {
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
