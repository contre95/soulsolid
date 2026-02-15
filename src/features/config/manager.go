package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Manager holds the application configuration and provides thread-safe access to it.
type Manager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
}

// processEnvVarNodes recursively processes YAML nodes to handle !env_var tags
func (m *Manager) processEnvVarNodes(node *yaml.Node, path []string) error {
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
			if err := m.processEnvVarNodes(node.Content[i], path); err != nil {
				return err
			}
		}
	} else if node.Kind == yaml.MappingNode {
		// Mapping nodes have key-value pairs: content[0]=key, content[1]=value, etc.
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode {
				newPath := append([]string{}, path...)
				newPath = append(newPath, keyNode.Value)
				if err := m.processEnvVarNodes(valueNode, newPath); err != nil {
					return err
				}
			}
		}
	} else if node.Kind == yaml.SequenceNode {
		// Sequence nodes: we don't add index to path as YAML paths don't include array indices
		for i := range node.Content {
			if err := m.processEnvVarNodes(node.Content[i], path); err != nil {
				return err
			}
		}
	}

	return nil
}

// loadConfig reads and parses a YAML configuration file.
func (m *Manager) loadConfig(path string) (*Config, error) {
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
	if err := m.processEnvVarNodes(&rootNode, []string{}); err != nil {
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

	return &cfg, nil
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

// newManager creates a new ConfigManager.
func NewManager(path string) (*Manager, error) {
	manager := &Manager{}
	manager.configPath = path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Info("Config file not found, creating default configuration", "path", path)
		if err := saveDefaultConfig(path, &defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		slog.Info("Default configuration created successfully", "path", path)
		manager := &Manager{config: &defaultConfig}
		if err := manager.EnsureDirectories(); err != nil {
			return nil, err
		}
		return manager, nil
	}
	// Create manager with nil config, load config
	cfg, err := manager.loadConfig(path)
	if err != nil {
		return nil, err
	}
	manager.config = cfg
	if err := manager.EnsureDirectories(); err != nil {
		return nil, err
	}
	return manager, nil
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
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Ensure the directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create config directory", "path", dir, "error", err)
		return err
	}

	file, err := os.Create(m.configPath)
	if err != nil {
		slog.Error("failed to create config file", "path", m.configPath, "error", err)
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(m.config); err != nil {
		slog.Error("failed to encode config", "path", m.configPath, "error", err)
		return err
	}

	slog.Info("Configuration saved successfully", "path", m.configPath)
	return nil
}

// EnsureDirectories creates the "library" and "download" directories if they don't exist.
func (m *Manager) EnsureDirectories() error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()
	if err := os.MkdirAll(cfg.LibraryPath, 0755); err != nil {
		return fmt.Errorf("failed to create library directory %s: %w", cfg.LibraryPath, err)
	}
	if err := os.MkdirAll(cfg.DownloadPath, 0755); err != nil {
		return fmt.Errorf("failed to create download directory %s: %w", cfg.DownloadPath, err)
	}
	slog.Info("Required directories created/verified", "library", cfg.LibraryPath, "downloads", cfg.DownloadPath)
	return nil
}

func (m *Manager) GetYAML() string {
	content, err := os.ReadFile(m.configPath)
	if err != nil {
		return "couldn't get config"
	}
	return string(content)
}

// GetEnabledMetadataProviders returns a map of enabled metadata providers
func (m *Manager) GetEnabledMetadataProviders() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	enabled := make(map[string]bool)
	if m.config.Metadata.Providers != nil {
		for name, provider := range m.config.Metadata.Providers {
			enabled[name] = provider.Enabled
		}
	}
	return enabled
}

// GetEnabledLyricsProviders returns a map of enabled lyrics providers
func (m *Manager) GetEnabledLyricsProviders() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	enabled := make(map[string]bool)
	if m.config.Lyrics.Providers != nil {
		for name, provider := range m.config.Lyrics.Providers {
			enabled[name] = provider.Enabled
		}
	}
	return enabled
}
