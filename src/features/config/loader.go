package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// setProviderSecret sets the secret for a provider from an environment variable
func setProviderSecret(cfg *Config, providerName, envVar string) {
	if key := os.Getenv(envVar); key != "" {
		if cfg.Metadata.Providers == nil {
			cfg.Metadata.Providers = make(map[string]Provider)
		}
		if provider, exists := cfg.Metadata.Providers[providerName]; exists {
			provider.Secret = &key
			cfg.Metadata.Providers[providerName] = provider
		} else {
			cfg.Metadata.Providers[providerName] = Provider{Enabled: false, Secret: &key}
		}
	}
}

// Load reads a YAML file from the given path and returns a new ConfigManager.
// If the file doesn't exist, creates a default configuration.
func Load(path string) (*Manager, error) {
	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Config file not found, creating default configuration", "path", path)
		defaultCfg := createDefaultConfig()

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

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set defaults for missing values

	// Override with environment variables if set
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}

	setProviderSecret(&cfg, "discogs", "DISCOGS_API_KEY")      // NOTE: Add this to the docs
	setProviderSecret(&cfg, "acoustid", "ACOUSTID_CLIENT_KEY") // NOTE: Add this to the docs

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
