package downloading

import (
	"fmt"
	"log/slog"
	"plugin"

	"github.com/contre95/soulsolid/src/features/config"
)

// PluginNewDownloaderFunc is the function signature that plugins must export
type PluginNewDownloaderFunc func(config map[string]interface{}) (Downloader, error)

// PluginManager manages loading and providing access to plugin downloaders
type PluginManager struct {
	downloaders map[string]Downloader
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		downloaders: make(map[string]Downloader),
	}
}

// LoadPlugins loads all configured plugins
func (pm *PluginManager) LoadPlugins(cfg *config.Config) error {
	slog.Info("Loading downloader plugins")

	// Load built-in downloaders
	// Demo downloader is loaded in main.go

	// Load plugin downloaders
	for _, pluginCfg := range cfg.Downloaders.Plugins {
		if err := pm.loadPlugin(pluginCfg); err != nil {
			slog.Error("Failed to load plugin", "name", pluginCfg.Name, "path", pluginCfg.Path, "error", err)
			// Continue loading other plugins
			continue
		}
	}

	slog.Info("Plugin loading completed", "total_downloaders", len(pm.downloaders))
	return nil
}

// loadPlugin loads a single plugin
func (pm *PluginManager) loadPlugin(pluginCfg config.PluginConfig) error {
	slog.Debug("Loading plugin", "name", pluginCfg.Name, "path", pluginCfg.Path)

	// Open the plugin
	p, err := plugin.Open(pluginCfg.Path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", pluginCfg.Path, err)
	}

	// Look up the NewDownloader symbol
	sym, err := p.Lookup("NewDownloader")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewDownloader function: %w", pluginCfg.Name, err)
	}

	// Assert the symbol is a function with the correct signature
	newDownloaderFunc, ok := sym.(func(map[string]interface{}) (Downloader, error))
	if !ok {
		return fmt.Errorf("plugin %s NewDownloader function has incorrect signature", pluginCfg.Name)
	}

	// Call the function to create the downloader
	downloader, err := newDownloaderFunc(pluginCfg.Config)
	if err != nil {
		return fmt.Errorf("failed to create downloader from plugin %s: %w", pluginCfg.Name, err)
	}

	// Store the downloader
	pm.downloaders[pluginCfg.Name] = downloader
	slog.Info("Successfully loaded plugin", "name", pluginCfg.Name, "downloader_name", downloader.Name())

	return nil
}

// GetDownloader returns a downloader by name
func (pm *PluginManager) GetDownloader(name string) (Downloader, bool) {
	downloader, exists := pm.downloaders[name]
	return downloader, exists
}

// AddDownloader adds a downloader to the manager
func (pm *PluginManager) AddDownloader(name string, downloader Downloader) {
	pm.downloaders[name] = downloader
}

// GetAllDownloaders returns all loaded downloaders
func (pm *PluginManager) GetAllDownloaders() map[string]Downloader {
	// Return a copy to prevent external modification
	result := make(map[string]Downloader)
	for k, v := range pm.downloaders {
		result[k] = v
	}
	return result
}

// GetDownloaderNames returns a list of all downloader names
func (pm *PluginManager) GetDownloaderNames() []string {
	names := make([]string, 0, len(pm.downloaders))
	for name := range pm.downloaders {
		names = append(names, name)
	}
	return names
}
