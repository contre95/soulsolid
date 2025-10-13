package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// setProviderAPIKey sets the API key for a provider from an environment variable
func setProviderAPIKey(cfg *Config, providerName, envVar string) {
	if key := os.Getenv(envVar); key != "" {
		if cfg.Tag.Providers == nil {
			cfg.Tag.Providers = make(map[string]Provider)
		}
		if provider, exists := cfg.Tag.Providers[providerName]; exists {
			provider.APIKey = key
			cfg.Tag.Providers[providerName] = provider
		} else {
			cfg.Tag.Providers[providerName] = Provider{Enabled: false, APIKey: key}
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
		return NewManager(defaultCfg), nil
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

	// Set defaults for missing values
	if cfg.Server.ViewsPath == "" {
		cfg.Server.ViewsPath = "./views"
	}

	// Override with environment variables if set
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}

	setProviderAPIKey(&cfg, "discogs", "DISCOGS_API_KEY")

	// Override views path with environment variable if set
	if viewsPath := os.Getenv("SS_VIEWS"); viewsPath != "" {
		cfg.Server.ViewsPath = viewsPath
	}

	return NewManager(&cfg), nil
}

// createDefaultConfig creates a new Config with sensible default values
func createDefaultConfig() *Config {
	return &Config{
		LibraryPath:  "./music",
		DownloadPath: "./downloads",
		Telegram: Telegram{
			Enabled:      false,
			Token:        "",
			AllowedUsers: []string{},
		},
		Logger: Logger{
			Enabled:   true,
			Level:     "info",
			Format:    "text",
			HTMXDebug: false,
		},
		Downloaders: Downloaders{
			Artwork: Artwork{
				Embedded: EmbeddedArtwork{
					Enabled: true,
					Size:    1000,
					Format:  "jpeg",
					Quality: 85,
				},
			},
		},
		Server: Server{
			PrintRoutes: false,
			Port:        3535,
			ViewsPath:   "./views",
		},
		Database: Database{
			Path: "./library.db",
		},
		Import: Import{
			Move:        false,
			AlwaysQueue: false,
			Duplicates:  "queue",
			PathOptions: Paths{
				Compilations:    "%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumSoundtrack: "%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumSingle:     "%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				AlbumEP:         "%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
				DefaultPath:     "%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			},
		},
		Tag: Tag{
			Providers: map[string]Provider{
				"discogs": {
					Enabled: false,
					APIKey:  "",
				},
				"musicbrainz": {
					Enabled: false,
				},
				"deezer": {
					Enabled: false,
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
