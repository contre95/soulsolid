package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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

	// Set defaults from createDefaultConfig
	defaultCfg := createDefaultConfig()
	// Convert default config to map and set defaults
	defaultMap := make(map[string]any)
	bytes, _ := yaml.Marshal(defaultCfg)
	yaml.Unmarshal(bytes, &defaultMap)
	for key, value := range defaultMap {
		v.SetDefault(key, value)
	}

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Config file not found, creating default configuration", "path", path)

		// Save default config to file
		if err := saveDefaultConfig(path, defaultCfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}

		slog.Info("Default configuration created successfully", "path", path)
		manager := NewManager(defaultCfg)
		if err := manager.EnsureDirectories(); err != nil {
			return nil, err
		}
		return manager, nil
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Merge indexed environment variables into slice fields
	mergeIndexedSlices(v, &cfg)

	// Validate
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	manager := NewManager(&cfg)
	if err := manager.EnsureDirectories(); err != nil {
		return nil, err
	}

	return manager, nil
}

// createDefaultConfig creates a new Config with sensible default values
func createDefaultConfig() *Config {
	return &Config{
		LibraryPath:  "./music",
		DownloadPath: "./downloads",
		Telegram: Telegram{
			Enabled:      false,
			Token:        "",                  // Can be obtained with https://t.me/BotFather
			AllowedUsers: []string{"user1"},   // No @
			BotHandle:    "@SoulsolidDemoBot", // With @
		},
		Logger: Logger{
			Enabled:   true,
			Level:     "info",
			Format:    "text",
			HTMXDebug: false,
		},
		Downloaders: Downloaders{
			Plugins: []PluginConfig{},
			Artwork: Artwork{
				Embedded: EmbeddedArtwork{
					Enabled: true,
					Size:    1000,
					Quality: 85,
				},
			},
		},
		Server: Server{
			PrintRoutes: false,
			Port:        3535,
		},
		Database: Database{
			Path: "./library.db",
		},

		Import: Import{
			Move:             false,
			AlwaysQueue:      false,
			Duplicates:       "queue",
			AutoStartWatcher: false,
			PathOptions: Paths{
				Compilations:    "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumSoundtrack: "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumSingle:     "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumEP:         "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				DefaultPath:     "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			},
		},
		Metadata: Metadata{
			Providers: map[string]Provider{
				"deezer": {
					Enabled: true,
				},
				"discogs": {
					Enabled: false,
					Secret:  nil,
				},
				"musicbrainz": {
					Enabled: true,
				},
				"acoustid": {
					Enabled: false,
					Secret:  nil,
				},
			},
		},
		Lyrics: Lyrics{
			Providers: map[string]LyricsProvider{
				"lrclib": {
					Enabled:      true,
					PreferSynced: false,
				},
			},
		},
		Sync: Sync{
			Enabled: false,
			Devices: []Device{},
		},
		Jobs: Jobs{
			Log:     true,
			LogPath: "./logs/jobs",
			Webhooks: WebhookConfig{
				Enabled:  false,
				JobTypes: []string{},
				Command:  "",
			},
		},
	}
}

// mergeIndexedSlices merges indexed environment variables into slice fields
func mergeIndexedSlices(v *viper.Viper, cfg *Config) {
	// Merge sync.devices
	deviceIndex := 0
	for {
		uuidKey := fmt.Sprintf("sync.devices.%d.uuid", deviceIndex)
		if !v.IsSet(uuidKey) {
			break
		}
		// Ensure slice is large enough
		if len(cfg.Sync.Devices) <= deviceIndex {
			cfg.Sync.Devices = append(cfg.Sync.Devices, Device{})
		}
		if v.IsSet(uuidKey) {
			cfg.Sync.Devices[deviceIndex].UUID = v.GetString(uuidKey)
		}
		if v.IsSet(fmt.Sprintf("sync.devices.%d.name", deviceIndex)) {
			cfg.Sync.Devices[deviceIndex].Name = v.GetString(fmt.Sprintf("sync.devices.%d.name", deviceIndex))
		}
		if v.IsSet(fmt.Sprintf("sync.devices.%d.sync_path", deviceIndex)) {
			cfg.Sync.Devices[deviceIndex].SyncPath = v.GetString(fmt.Sprintf("sync.devices.%d.sync_path", deviceIndex))
		}
		deviceIndex++
	}

	// Merge telegram.allowedUsers (indexed)
	userIndex := 0
	hasIndexedUsers := false
	for {
		userKey := fmt.Sprintf("telegram.allowedUsers.%d", userIndex)
		if !v.IsSet(userKey) {
			break
		}
		hasIndexedUsers = true
		if len(cfg.Telegram.AllowedUsers) <= userIndex {
			cfg.Telegram.AllowedUsers = append(cfg.Telegram.AllowedUsers, "")
		}
		cfg.Telegram.AllowedUsers[userIndex] = v.GetString(userKey)
		userIndex++
	}

	// If no indexed users, check for comma-separated string
	if !hasIndexedUsers {
		if allowedUsersStr := v.GetString("telegram.allowedUsers"); allowedUsersStr != "" && strings.Contains(allowedUsersStr, ",") {
			users := strings.Split(allowedUsersStr, ",")
			for i, user := range users {
				users[i] = strings.TrimSpace(user)
			}
			cfg.Telegram.AllowedUsers = users
		}
	}

	// Note: jobs.webhooks.job_types could also be indexed, but we'll rely on comma-separated or YAML
}

// saveDefaultConfig saves the default configuration to the specified file path
func saveDefaultConfig(path string, cfg *Config) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}
	slog.Info("Default configuration saved", "path", path)
	return nil
}
