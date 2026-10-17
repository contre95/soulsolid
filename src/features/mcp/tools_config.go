package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Service) registerConfigTools(srv *server.MCPServer) {
	s.addReadTool(srv, mcpgo.NewTool("get_config",
		mcpgo.WithDescription("Get the current SoulSolid configuration. Secrets (tokens, provider secrets, downloader plugin config, webhook command) are redacted."),
	), s.getConfig)

	s.addDestructiveTool(srv, mcpgo.NewTool("update_settings",
		mcpgo.WithDescription("Update one or more SoulSolid settings and persist them to config.yaml. Pass a JSON object of dot-path keys to string values, e.g. {\"import.duplicates\":\"skip\",\"logger.level\":\"debug\"}. Supported keys: import.move, import.always_queue, import.duplicates, import.allow_missing_metadata.{artist,album,title,year,genre}, import.paths.{default_path,compilations,album:soundtrack,album:single,album:ep,fat32_safe}, telegram.{enabled,bot_handle}, metadata.providers.<name>.enabled, lyrics.providers.<name>.{enabled,prefer_synced}, logger.{enabled,level,format,htmx_debug}, jobs.log. Boolean values as \"true\"/\"false\". Paths, secrets and access lists (libraryPath, downloadPath, jobs.log_path, telegram token/allowedUsers, provider secrets) can only be changed in the config file or the web UI."),
		mcpgo.WithString("settings", mcpgo.Required(), mcpgo.Description("JSON object of setting key/value pairs to update")),
	), s.updateSettings)
}

func (s *Service) getConfig(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	return jsonResult(redactConfig(s.config.Get())), nil
}

const redacted = "[redacted]"

// redactConfig returns a copy of cfg with every secret masked. Tool output
// lands in the agent's context (and often its provider's logs), so credentials
// must never leave through it. Maps and slices are cloned before masking so the
// live config is never touched.
func redactConfig(cfg *config.Config) config.Config {
	out := *cfg
	if out.Telegram.Token != "" {
		out.Telegram.Token = redacted
	}
	if out.MCP.Token != "" {
		out.MCP.Token = redacted
	}
	if out.Jobs.Webhooks.Command != "" {
		out.Jobs.Webhooks.Command = redacted
	}
	out.Metadata.Providers = maps.Clone(out.Metadata.Providers)
	for name, p := range out.Metadata.Providers {
		if p.Secret != nil && *p.Secret != "" {
			masked := redacted
			p.Secret = &masked
			out.Metadata.Providers[name] = p
		}
	}
	// Plugin config is free-form and typically holds credentials (ARLs, API keys).
	out.Downloaders.Plugins = slices.Clone(out.Downloaders.Plugins)
	for i, p := range out.Downloaders.Plugins {
		if len(p.Config) == 0 {
			continue
		}
		masked := make(map[string]any, len(p.Config))
		for k := range p.Config {
			masked[k] = redacted
		}
		out.Downloaders.Plugins[i].Config = masked
	}
	return out
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
	case "import.duplicates":
		if value != "replace" && value != "skip" && value != "queue" {
			return fmt.Errorf("import.duplicates must be one of: replace, skip, queue")
		}
		cfg.Import.Duplicates = value
		return nil
	case "import.paths.default_path", "import.paths.compilations", "import.paths.album:soundtrack", "import.paths.album:single", "import.paths.album:ep":
		tmpl, err := validatePathTemplate(key, value)
		if err != nil {
			return err
		}
		switch key {
		case "import.paths.default_path":
			cfg.Import.PathOptions.DefaultPath = tmpl
		case "import.paths.compilations":
			cfg.Import.PathOptions.Compilations = tmpl
		case "import.paths.album:soundtrack":
			cfg.Import.PathOptions.AlbumSoundtrack = tmpl
		case "import.paths.album:single":
			cfg.Import.PathOptions.AlbumSingle = tmpl
		case "import.paths.album:ep":
			cfg.Import.PathOptions.AlbumEP = tmpl
		}
		return nil
	case "telegram.bot_handle":
		cfg.Telegram.BotHandle = value
		return nil
	case "logger.level":
		if !slices.Contains([]string{"debug", "info", "warn", "error"}, value) {
			return fmt.Errorf("logger.level must be one of: debug, info, warn, error")
		}
		cfg.Logger.Level = value
		return nil
	case "logger.format":
		if !slices.Contains([]string{"text", "json", "logfmt"}, value) {
			return fmt.Errorf("logger.format must be one of: text, json, logfmt")
		}
		cfg.Logger.Format = value
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

// validatePathTemplate rejects templates that would render files outside the
// library: empty ones (straight into the root, mirroring the UI's check — except
// compilations, which the UI also allows empty), absolute paths, and ".." segments.
func validatePathTemplate(key, value string) (string, error) {
	tmpl := strings.TrimSpace(value)
	if tmpl == "" {
		if key == "import.paths.compilations" {
			return "", nil
		}
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	if filepath.IsAbs(tmpl) {
		return "", fmt.Errorf("%s must be relative to the library path", key)
	}
	for seg := range strings.SplitSeq(filepath.ToSlash(tmpl), "/") {
		if seg == ".." {
			return "", fmt.Errorf("%s must not contain '..' segments", key)
		}
	}
	return tmpl, nil
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
