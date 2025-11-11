package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Manager holds the application configuration and provides thread-safe access to it.
type Manager struct {
	mu     sync.RWMutex
	config *Config
}

// NewManager creates a new ConfigManager.
func NewManager(config *Config) *Manager {
	return &Manager{config: config}
}

// Get returns the current configuration.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Update updates the configuration.
func (m *Manager) Update(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldConfig := m.config
	m.config = config

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

	file, err := os.Create(path)
	if err != nil {
		slog.Error("failed to create config file", "path", path, "error", err)
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(m.config); err != nil {
		slog.Error("failed to encode config", "path", path, "error", err)
		return err
	}

	slog.Info("Configuration saved successfully", "path", path)
	return nil
}

// EnsureDirectories creates the library and download directories if they don't exist.
func (m *Manager) EnsureDirectories() error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

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

// redactedCfg gets a redacted copy of the Config
func (m *Manager) redactedCfg() Config {
	var cfgCpy = *m.Get()
	cfgCpy.Telegram.Token = "<redacted>"
	// Note: DownloadPath is not redacted as it's a path, not a secret
	return cfgCpy
}

// GetJSON returns the current configuration as a JSON string.
func (m *Manager) GetJSON() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	jsonBytes, err := json.Marshal(m.redactedCfg())
	if err != nil {
		slog.Error("failed to marshal config to JSON", "error", err)
		return err.Error()
	}
	return string(jsonBytes)
}

func (m *Manager) GetYAML() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	yamlBytes, err := yaml.Marshal(m.redactedCfg())
	if err != nil {
		slog.Error("failed to marshal config to YAML", "error", err)
		return err.Error()
	}
	return string(yamlBytes)
}
