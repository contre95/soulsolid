package downloading

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"plugin"
	"strings"
	"sync"

	"github.com/contre95/soulsolid/src/features/config"
)

// PluginNewDownloaderFunc is the function signature that plugins must export
type PluginNewDownloaderFunc func(config map[string]interface{}) (Downloader, error)

// PluginManager manages loading and providing access to plugin downloaders
type PluginManager struct {
	downloaders map[string]Downloader
	mu          sync.RWMutex
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

	for _, pluginCfg := range cfg.Downloaders.Plugins {
		if err := pm.loadPlugin(pluginCfg); err != nil {
			slog.Error("Failed to load plugin", "name", pluginCfg.Name, "path", pluginCfg.Path, "error", err)
			continue
		}
	}

	pm.mu.RLock()
	total := len(pm.downloaders)
	pm.mu.RUnlock()
	slog.Info("Plugin loading completed", "total_downloaders", total)
	return nil
}

// loadPlugin loads a single plugin
func (pm *PluginManager) loadPlugin(pluginCfg config.PluginConfig) error {
	slog.Debug("Loading plugin", "name", pluginCfg.Name, "path", pluginCfg.Path)

	pluginPath := pluginCfg.Path

	// If path is a URL, download the plugin to a temporary file
	if strings.HasPrefix(pluginCfg.Path, "http://") || strings.HasPrefix(pluginCfg.Path, "https://") {
		tempFile, err := os.CreateTemp("", "*.so")
		if err != nil {
			return fmt.Errorf("failed to create temp file for plugin %s: %w", pluginCfg.Name, err)
		}
		defer tempFile.Close()

		resp, err := http.Get(pluginCfg.Path)
		if err != nil {
			os.Remove(tempFile.Name()) // cleanup on error
			return fmt.Errorf("failed to download plugin %s from %s: %w", pluginCfg.Name, pluginCfg.Path, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			os.Remove(tempFile.Name()) // cleanup on error
			return fmt.Errorf("failed to download plugin %s: HTTP %d", pluginCfg.Name, resp.StatusCode)
		}

		_, err = io.Copy(tempFile, resp.Body)
		if err != nil {
			os.Remove(tempFile.Name()) // cleanup on error
			return fmt.Errorf("failed to write plugin %s to temp file: %w", pluginCfg.Name, err)
		}

		tempFile.Close() // close before opening as plugin
		pluginPath = tempFile.Name()
	}

	p, err := plugin.Open(pluginPath)
	if err != nil {
		if strings.HasPrefix(pluginCfg.Path, "http://") || strings.HasPrefix(pluginCfg.Path, "https://") {
			os.Remove(pluginPath) // cleanup temp file on error
		}
		return fmt.Errorf("failed to open plugin %s: %w", pluginCfg.Path, err)
	}

	sym, err := p.Lookup("NewDownloader")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewDownloader function: %w", pluginCfg.Name, err)
	}

	newDownloaderFunc, ok := sym.(func(map[string]any) (Downloader, error))
	if !ok {
		return fmt.Errorf("plugin %s NewDownloader function has incorrect signature", pluginCfg.Name)
	}

	downloader, err := newDownloaderFunc(pluginCfg.Config)
	// NOTE: When debuggin, take into account that plugins might override this custom pluginCfg.Config with custom configurations with environment variables given that this config might contain secret values to connect to their respective providers.
	if err != nil {
		return fmt.Errorf("failed to create downloader from plugin %s: %w", pluginCfg.Name, err)
	}

	pm.mu.Lock()
	pm.downloaders[pluginCfg.Name] = downloader
	total := len(pm.downloaders)
	pm.mu.Unlock()
	slog.Info("Successfully loaded plugin", "name", pluginCfg.Name, "downloader_name", downloader.Name(), "total_downloaders", total)

	return nil
}

// GetDownloader returns a downloader by name
func (pm *PluginManager) GetDownloader(name string) (Downloader, bool) {
	pm.mu.RLock()
	downloader, exists := pm.downloaders[name]
	pm.mu.RUnlock()
	return downloader, exists
}

// AddDownloader adds a downloader to the manager
func (pm *PluginManager) AddDownloader(name string, downloader Downloader) {
	pm.mu.Lock()
	pm.downloaders[name] = downloader
	pm.mu.Unlock()
}

// GetAllDownloaders returns all loaded downloaders
func (pm *PluginManager) GetAllDownloaders() map[string]Downloader {
	pm.mu.RLock()
	result := make(map[string]Downloader)
	maps.Copy(result, pm.downloaders)
	pm.mu.RUnlock()
	return result
}

// GetDownloaderNames returns a list of all downloader names
func (pm *PluginManager) GetDownloaderNames() []string {
	pm.mu.RLock()
	names := make([]string, 0, len(pm.downloaders))
	for name := range pm.downloaders {
		names = append(names, name)
	}
	pm.mu.RUnlock()
	return names
}
