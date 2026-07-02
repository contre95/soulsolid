package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerConfigTools(srv *server.MCPServer) {
	srv.AddTool(mcpgo.NewTool("get_config",
		mcpgo.WithDescription("Get the current SoulSolid configuration"),
	), s.getConfig)

	srv.AddTool(mcpgo.NewTool("update_settings",
		mcpgo.WithDescription("Update one or more SoulSolid settings and persist them to config.yaml. Pass a JSON object of dot-path keys to string values, e.g. {\"import.duplicates\":\"skip\",\"logger.level\":\"debug\"}. Supported keys: libraryPath, downloadPath, import.move, import.always_queue, import.duplicates, import.allow_missing_metadata.{artist,album,title,year,genre}, import.paths.{default_path,compilations,album:soundtrack,album:single,album:ep,fat32_safe}, telegram.{enabled,token,allowedUsers,bot_handle}, metadata.providers.<name>.{enabled,secret}, lyrics.providers.<name>.{enabled,prefer_synced}, logger.{enabled,level,format,htmx_debug}, jobs.{log,log_path}. Boolean values as \"true\"/\"false\"."),
		mcpgo.WithString("settings", mcpgo.Required(), mcpgo.Description("JSON object of setting key/value pairs to update")),
	), s.updateSettings)
}

func (s *Service) getConfig(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return jsonResult(s.config.Get()), nil
}

func (s *Service) updateSettings(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	raw := strArg(req.Params.Arguments, "settings")
	var fields map[string]string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return errResult("settings must be a JSON object of string key/value pairs: " + err.Error()), nil
	}
	if len(fields) == 0 {
		return errResult("no settings provided"), nil
	}

	// Work on a copy so partial failures never leave a half-applied config,
	// and clone the provider maps the shallow copy would otherwise share.
	newConfig := *s.config.Get()
	newConfig.Metadata.Providers = maps.Clone(newConfig.Metadata.Providers)
	newConfig.Lyrics.Providers = maps.Clone(newConfig.Lyrics.Providers)

	for key, value := range fields {
		if err := applySetting(&newConfig, key, value); err != nil {
			return errResult(err.Error()), nil
		}
	}

	s.config.Update(&newConfig)
	if err := s.config.Save(); err != nil {
		slog.Warn("failed to save config to file (this is normal in containerized environments)", "error", err)
		return textResult("settings updated in memory, but could not be saved to file: " + err.Error()), nil
	}
	return textResult("settings updated and saved successfully"), nil
}

// applySetting applies a single dot-path setting to cfg. It mirrors the fields
// the UI settings form exposes (config.Handler.UpdateSettings).
func applySetting(cfg *config.Config, key, value string) error {
	parseBool := func() (bool, error) {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("setting %q expects a boolean, got %q", key, value)
		}
		return b, nil
	}

	var boolTarget *bool
	switch key {
	case "libraryPath":
		cfg.LibraryPath = value
		return nil
	case "downloadPath":
		cfg.DownloadPath = value
		return nil
	case "import.duplicates":
		if value != "replace" && value != "skip" && value != "queue" {
			return fmt.Errorf("import.duplicates must be one of: replace, skip, queue")
		}
		cfg.Import.Duplicates = value
		return nil
	case "import.paths.default_path":
		cfg.Import.PathOptions.DefaultPath = value
		return nil
	case "import.paths.compilations":
		cfg.Import.PathOptions.Compilations = value
		return nil
	case "import.paths.album:soundtrack":
		cfg.Import.PathOptions.AlbumSoundtrack = value
		return nil
	case "import.paths.album:single":
		cfg.Import.PathOptions.AlbumSingle = value
		return nil
	case "import.paths.album:ep":
		cfg.Import.PathOptions.AlbumEP = value
		return nil
	case "telegram.token":
		cfg.Telegram.Token = value
		return nil
	case "telegram.bot_handle":
		cfg.Telegram.BotHandle = value
		return nil
	case "telegram.allowedUsers":
		var users []string
		for u := range strings.SplitSeq(value, ",") {
			if trimmed := strings.TrimSpace(u); trimmed != "" {
				users = append(users, trimmed)
			}
		}
		cfg.Telegram.AllowedUsers = users
		return nil
	case "logger.level":
		cfg.Logger.Level = value
		return nil
	case "logger.format":
		cfg.Logger.Format = value
		return nil
	case "jobs.log_path":
		cfg.Jobs.LogPath = value
		return nil
	case "import.move":
		boolTarget = &cfg.Import.Move
	case "import.always_queue":
		boolTarget = &cfg.Import.AlwaysQueue
	case "import.allow_missing_metadata.artist":
		boolTarget = &cfg.Import.AllowMissingMetadata.Artist
	case "import.allow_missing_metadata.album":
		boolTarget = &cfg.Import.AllowMissingMetadata.Album
	case "import.allow_missing_metadata.title":
		boolTarget = &cfg.Import.AllowMissingMetadata.Title
	case "import.allow_missing_metadata.year":
		boolTarget = &cfg.Import.AllowMissingMetadata.Year
	case "import.allow_missing_metadata.genre":
		boolTarget = &cfg.Import.AllowMissingMetadata.Genre
	case "import.paths.fat32_safe":
		boolTarget = &cfg.Import.PathOptions.Fat32Safe
	case "telegram.enabled":
		boolTarget = &cfg.Telegram.Enabled
	case "logger.enabled":
		boolTarget = &cfg.Logger.Enabled
	case "logger.htmx_debug":
		boolTarget = &cfg.Logger.HTMXDebug
	case "jobs.log":
		boolTarget = &cfg.Jobs.Log
	default:
		// metadata.providers.<name>.enabled|secret
		if name, field, ok := providerKey(key, "metadata.providers."); ok {
			provider := cfg.Metadata.Providers[name]
			switch field {
			case "enabled":
				b, err := parseBool()
				if err != nil {
					return err
				}
				provider.Enabled = b
			case "secret":
				secret := value
				provider.Secret = &secret
			default:
				return fmt.Errorf("unknown metadata provider setting: %s", key)
			}
			cfg.Metadata.Providers[name] = provider
			return nil
		}
		// lyrics.providers.<name>.enabled|prefer_synced
		if name, field, ok := providerKey(key, "lyrics.providers."); ok {
			provider := cfg.Lyrics.Providers[name]
			switch field {
			case "enabled":
				b, err := parseBool()
				if err != nil {
					return err
				}
				provider.Enabled = b
			case "prefer_synced":
				b, err := parseBool()
				if err != nil {
					return err
				}
				provider.PreferSynced = b
			default:
				return fmt.Errorf("unknown lyrics provider setting: %s", key)
			}
			cfg.Lyrics.Providers[name] = provider
			return nil
		}
		return fmt.Errorf("unknown or unsupported setting: %s", key)
	}

	b, err := parseBool()
	if err != nil {
		return err
	}
	*boolTarget = b
	return nil
}

// providerKey splits "prefix<name>.<field>" into its name and field parts.
func providerKey(key, prefix string) (name, field string, ok bool) {
	rest, found := strings.CutPrefix(key, prefix)
	if !found {
		return "", "", false
	}
	name, field, found = strings.Cut(rest, ".")
	if !found || name == "" || field == "" {
		return "", "", false
	}
	return name, field, true
}
