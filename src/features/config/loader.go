package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// processEnvVarNodes recursively processes YAML nodes to handle !env_var tags
func processEnvVarNodes(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	// Check if this node has a !env_var tag
	if node.Tag == "!env_var" {
		if node.Kind != yaml.ScalarNode {
			return fmt.Errorf("!env_var tag can only be used with scalar values (line %d)", node.Line)
		}

		// Extract environment variable name
		envVarName := strings.TrimSpace(node.Value)
		if envVarName == "" {
			return fmt.Errorf("!env_var tag requires environment variable name (line %d)", node.Line)
		}

		// Get environment variable value
		value := os.Getenv(envVarName)
		if value == "" {
			return fmt.Errorf("environment variable %s is not set or empty (referenced at line %d)", envVarName, node.Line)
		}

		// Replace the node value with the environment variable value
		node.Tag = ""
		node.Value = value
		return nil
	}

	// Recursively process child nodes
	// Start with DocumentNode content if present
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		for i := range node.Content {
			if err := processEnvVarNodes(node.Content[i]); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
		for i := range node.Content {
			if err := processEnvVarNodes(node.Content[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// setProviderSecret sets the secret for a provider from an environment variable
func setProviderSecret(cfg *Config, providerName, envVar string) {
	if key := os.Getenv(envVar); key != "" {
		slog.Warn("DEPRECATED: Using environment variable override for provider secret. Migrate to !env_var syntax in config.yaml",
			"provider", providerName, "env_var", envVar)
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

	// Read the entire file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse YAML into a Node first
	var rootNode yaml.Node
	if err := yaml.Unmarshal(content, &rootNode); err != nil {
		return nil, err
	}

	// Process !env_var tags in the node tree
	if err := processEnvVarNodes(&rootNode); err != nil {
		return nil, err
	}

	// Decode the processed node tree into Config struct
	var cfg Config
	if err := rootNode.Decode(&cfg); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Set defaults for missing values

	// Override with environment variables if set (deprecated)
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		slog.Warn("DEPRECATED: Using TELEGRAM_TOKEN environment variable override. Migrate to !env_var syntax in config.yaml")
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
			Token:        "",                                   // Can be obtained with https://t.me/BotFather
			AllowedUsers: []string{"<your_telegram_username>"}, // No @
			BotHandle:    "@<YourTelegramUserBot>",             // With @
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
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

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
