package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Load reads a YAML file from the given path and returns a new ConfigManager.
// If the file doesn't exist, creates a default configuration.
func Load(path string) (*Manager, error) {
	v := viper.New()

	// Configure Viper
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("SS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", ":", "_"))
	v.AutomaticEnv() // Automatically bind environment variables with SS_ prefix

	// Set defaults using viper.SetDefault
	setViperDefaults(v)

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Config file not found, creating default configuration", "path", path)

		// Write default config to file using viper
		if err := v.SafeWriteConfigAs(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		slog.Info("Default configuration created successfully", "path", path)
		manager := NewManager(v)
		if err := manager.EnsureDirectories(); err != nil {
			return nil, err
		}
		return manager, nil
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Merge indexed environment variables into slice fields in viper
	mergeIndexedSlicesIntoViper(v)

	// Unmarshal config for validation
	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Create manager with viper instance
	manager := NewManager(v)
	if err := manager.EnsureDirectories(); err != nil {
		return nil, err
	}

	return manager, nil
}

// setViperDefaults sets default configuration values using viper.SetDefault
func setViperDefaults(v *viper.Viper) {
	v.SetDefault("libraryPath", "./music")
	v.SetDefault("downloadPath", "./downloads")
	v.SetDefault("telegram.enabled", false)
	v.SetDefault("telegram.token", "")
	v.SetDefault("telegram.allowedUsers", []string{"user1"})
	v.SetDefault("telegram.botHandle", "@SoulsolidDemoBot")
	v.SetDefault("logger.enabled", true)
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "text")
	v.SetDefault("logger.htmxDebug", false)
	v.SetDefault("downloaders.plugins", []PluginConfig{})
	v.SetDefault("downloaders.artwork.embedded.enabled", true)
	v.SetDefault("downloaders.artwork.embedded.size", 1000)
	v.SetDefault("downloaders.artwork.embedded.quality", 85)
	v.SetDefault("server.printRoutes", false)
	v.SetDefault("server.port", 3535)
	v.SetDefault("database.path", "./library.db")
	v.SetDefault("import.move", false)
	v.SetDefault("import.alwaysQueue", false)
	v.SetDefault("import.duplicates", "queue")
	v.SetDefault("import.autoStartWatcher", false)
	v.SetDefault("import.pathOptions.compilations", "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}")
	v.SetDefault("import.pathOptions.album:soundtrack", "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}")
	v.SetDefault("import.pathOptions.album:single", "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}")
	v.SetDefault("import.pathOptions.album:ep", "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}")
	v.SetDefault("import.pathOptions.defaultPath", "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}")
	v.SetDefault("metadata.providers.deezer.enabled", true)
	v.SetDefault("metadata.providers.discogs.enabled", false)
	v.SetDefault("metadata.providers.discogs.secret", nil)
	v.SetDefault("metadata.providers.musicbrainz.enabled", true)
	v.SetDefault("metadata.providers.acoustid.enabled", false)
	v.SetDefault("metadata.providers.acoustid.secret", nil)
	v.SetDefault("lyrics.providers.lrclib.enabled", true)
	v.SetDefault("lyrics.providers.lrclib.preferSynced", false)
	v.SetDefault("sync.enabled", false)
	v.SetDefault("sync.devices", []Device{})
	v.SetDefault("jobs.log", true)
	v.SetDefault("jobs.logPath", "./logs/jobs")
	v.SetDefault("jobs.webhooks.enabled", false)
	v.SetDefault("jobs.webhooks.jobTypes", []string{})
	v.SetDefault("jobs.webhooks.command", "")
}

// mergeIndexedSlicesIntoViper merges indexed environment variables into slice fields in viper.
func mergeIndexedSlicesIntoViper(v *viper.Viper) {
	// Merge sync.devices
	var devices []Device
	deviceIndex := 0
	for {
		uuidKey := fmt.Sprintf("sync.devices.%d.uuid", deviceIndex)
		if !v.IsSet(uuidKey) {
			break
		}
		device := Device{}
		if v.IsSet(uuidKey) {
			device.UUID = v.GetString(uuidKey)
		}
		if v.IsSet(fmt.Sprintf("sync.devices.%d.name", deviceIndex)) {
			device.Name = v.GetString(fmt.Sprintf("sync.devices.%d.name", deviceIndex))
		}
		if v.IsSet(fmt.Sprintf("sync.devices.%d.sync_path", deviceIndex)) {
			device.SyncPath = v.GetString(fmt.Sprintf("sync.devices.%d.sync_path", deviceIndex))
		}
		devices = append(devices, device)
		deviceIndex++
	}
	if len(devices) > 0 {
		v.Set("sync.devices", devices)
	}

	// Merge telegram.allowedUsers (indexed)
	var users []string
	userIndex := 0
	hasIndexedUsers := false
	for {
		userKey := fmt.Sprintf("telegram.allowedUsers.%d", userIndex)
		if !v.IsSet(userKey) {
			break
		}
		hasIndexedUsers = true
		users = append(users, v.GetString(userKey))
		userIndex++
	}
	// If no indexed users, check for comma-separated string
	if !hasIndexedUsers {
		if allowedUsersStr := v.GetString("telegram.allowedUsers"); allowedUsersStr != "" && strings.Contains(allowedUsersStr, ",") {
			users = strings.Split(allowedUsersStr, ",")
			for i, user := range users {
				users[i] = strings.TrimSpace(user)
			}
		}
	}
	if len(users) > 0 {
		v.Set("telegram.allowedUsers", users)
	}
}
