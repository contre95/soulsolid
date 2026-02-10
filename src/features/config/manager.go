package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Manager holds the application configuration and provides thread-safe access to it.
type Manager struct {
	mu sync.RWMutex
	v  *viper.Viper // viper instance holding configuration
}

// NewManager creates a new ConfigManager from a viper instance.
func NewManager(v *viper.Viper) *Manager {
	return &Manager{v: v}
}

// getConfigUnsafe returns the current configuration without locking (internal use).
func (m *Manager) getConfigUnsafe() (*Config, error) {
	var cfg Config
	if err := m.v.Unmarshal(&cfg, viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Get returns the current configuration.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, err := m.getConfigUnsafe()
	if err != nil {
		slog.Error("failed to unmarshal config", "error", err)
		// Return empty config as fallback
		return &Config{}
	}
	return cfg
}

// configToMap converts a Config to a map[string]any using YAML marshaling.
func configToMap(cfg *Config) (map[string]any, error) {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := yaml.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Update updates the configuration.
func (m *Manager) Update(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get old config for logging
	oldConfig, _ := m.getConfigUnsafe()

	// Convert new config to map and set in viper
	configMap, err := configToMap(config)
	if err != nil {
		slog.Error("failed to convert config to map", "error", err)
		return
	}
	for key, value := range configMap {
		m.v.Set(key, value)
	}

	// Log configuration changes
	if oldConfig != nil {
		slog.Debug("Configuration updated",
			"library_path_changed", oldConfig.LibraryPath != config.LibraryPath,
			"import_move_changed", oldConfig.Import.Move != config.Import.Move,
			"import_always_queue_changed", oldConfig.Import.AlwaysQueue != config.Import.AlwaysQueue,
			"telegram_enabled_changed", oldConfig.Telegram.Enabled != config.Telegram.Enabled,
			"logger_enabled_changed", oldConfig.Logger.Enabled != config.Logger.Enabled,
		)
	}
}

// Save writes the current configuration to the specified file path.
func (m *Manager) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Temporarily set config file path and write
	m.v.SetConfigFile(path)
	if err := m.v.WriteConfigAs(path); err != nil {
		slog.Error("failed to write config file", "path", path, "error", err)
		return err
	}

	slog.Info("Configuration saved successfully", "path", path)
	return nil
}

// EnsureDirectories creates the library and download directories if they don't exist.
func (m *Manager) EnsureDirectories() error {
	m.mu.RLock()
	cfg, err := m.getConfigUnsafe()
	m.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Create library directory
	if err := os.MkdirAll(cfg.LibraryPath, 0755); err != nil {
		return fmt.Errorf("failed to create library directory %s: %w", cfg.LibraryPath, err)
	}

	// Create download directory
	if err := os.MkdirAll(cfg.DownloadPath, 0755); err != nil {
		return fmt.Errorf("failed to create download directory %s: %w", cfg.DownloadPath, err)
	}

	slog.Info("Required directories created/verified", "library", cfg.LibraryPath, "downloads", cfg.DownloadPath)
	return nil
}

// redactConfig returns a redacted copy of the Config
func redactConfig(cfg *Config) Config {
	var cfgCpy = *cfg
	cfgCpy.Telegram.Token = "<redacted>"
	// Note: DownloadPath is not redacted as it's a path, not a secret
	return cfgCpy
}

// GetJSON returns the current configuration as a JSON string.
func (m *Manager) GetJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, err := m.getConfigUnsafe()
	if err != nil {
		slog.Error("failed to unmarshal config for JSON", "error", err)
		return err.Error()
	}
	redacted := redactConfig(cfg)
	jsonBytes, err := json.Marshal(redacted)
	if err != nil {
		slog.Error("failed to marshal config to JSON", "error", err)
		return err.Error()
	}
	return string(jsonBytes)
}

func (m *Manager) GetYAML() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, err := m.getConfigUnsafe()
	if err != nil {
		slog.Error("failed to unmarshal config for YAML", "error", err)
		return err.Error()
	}
	redacted := redactConfig(cfg)
	yamlBytes, err := yaml.Marshal(redacted)
	if err != nil {
		slog.Error("failed to marshal config to YAML", "error", err)
		return err.Error()
	}
	return string(yamlBytes)
}

// GetEnabledMetadataProviders returns a map of enabled metadata providers
func (m *Manager) GetEnabledMetadataProviders() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, err := m.getConfigUnsafe()
	if err != nil {
		slog.Error("failed to unmarshal config for metadata providers", "error", err)
		return make(map[string]bool)
	}
	enabled := make(map[string]bool)
	if cfg.Metadata.Providers != nil {
		for name, provider := range cfg.Metadata.Providers {
			enabled[name] = provider.Enabled
		}
	}
	return enabled
}

// GetEnabledLyricsProviders returns a map of enabled lyrics providers
func (m *Manager) GetEnabledLyricsProviders() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, err := m.getConfigUnsafe()
	if err != nil {
		slog.Error("failed to unmarshal config for lyrics providers", "error", err)
		return make(map[string]bool)
	}
	enabled := make(map[string]bool)
	if cfg.Lyrics.Providers != nil {
		for name, provider := range cfg.Lyrics.Providers {
			enabled[name] = provider.Enabled
		}
	}
	return enabled
}
