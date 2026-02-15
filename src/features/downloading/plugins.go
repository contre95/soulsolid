package downloading

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
)

// PluginNewDownloaderFunc is the function signature that plugins must export
type PluginNewDownloaderFunc func(config map[string]interface{}) (Downloader, error)

// PluginManager manages loading and providing access to plugin downloaders
type PluginManager struct {
	downloaders map[string]Downloader
	mu          sync.RWMutex
}

// findModuleRoot finds the root directory of the current Go module by looking for go.mod
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found in current or parent directories")
}

// buildFromGit builds a plugin from a git repository URL
func buildFromGit(url string) (string, error) {
	// Create temporary directory for cloning and building
	tempDir, err := os.MkdirTemp("", "soulsolid-plugin-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clean up temp directory on error
	var buildErr error
	defer func() {
		if buildErr != nil {
			os.RemoveAll(tempDir)
		}
	}()

	slog.Debug("Cloning git repository", "url", url, "tempDir", tempDir)

	// Clone repository
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, tempDir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		buildErr = fmt.Errorf("failed to clone repository: %w: %s", err, stderr.String())
		return "", buildErr
	}

	// Find module root of soulsolid
	moduleRoot, err := findModuleRoot()
	if err != nil {
		buildErr = fmt.Errorf("failed to find soulsolid module root: %w", err)
		return "", buildErr
	}

	pluginDir := tempDir

	// Check if go.mod exists in plugin directory
	goModPath := filepath.Join(pluginDir, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		buildErr = fmt.Errorf("plugin directory does not contain go.mod: %w", err)
		return "", buildErr
	}

	// Add replace directive for soulsolid module
	cmd = exec.Command("go", "mod", "edit", "-replace=github.com/contre95/soulsolid="+moduleRoot)
	cmd.Dir = pluginDir
	if output, err := cmd.CombinedOutput(); err != nil {
		buildErr = fmt.Errorf("failed to add replace directive: %w: %s", err, output)
		return "", buildErr
	}

	// Run go mod tidy
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = pluginDir
	if output, err := cmd.CombinedOutput(); err != nil {
		buildErr = fmt.Errorf("failed to run go mod tidy: %w: %s", err, output)
		return "", buildErr
	}

	// Build the plugin
	outputFile := filepath.Join(tempDir, "plugin.so")
	cmd = exec.Command("go", "build", "-buildmode=plugin", "-o", outputFile, ".")
	cmd.Dir = pluginDir
	if output, err := cmd.CombinedOutput(); err != nil {
		buildErr = fmt.Errorf("failed to build plugin: %w: %s", err, output)
		return "", buildErr
	}

	slog.Debug("Successfully built plugin from git", "url", url, "output", outputFile)

	// Move .so file to a new temporary file (outside the source directory)
	// so we can clean up the source directory
	pluginSo, err := os.CreateTemp("", "*.so")
	if err != nil {
		buildErr = fmt.Errorf("failed to create temp .so file: %w", err)
		return "", buildErr
	}
	pluginSo.Close()

	if err := os.Rename(outputFile, pluginSo.Name()); err != nil {
		// If rename fails (cross-device), copy instead
		src, err := os.Open(outputFile)
		if err != nil {
			buildErr = fmt.Errorf("failed to open built plugin: %w", err)
			return "", buildErr
		}
		defer src.Close()

		dst, err := os.Create(pluginSo.Name())
		if err != nil {
			buildErr = fmt.Errorf("failed to create destination .so file: %w", err)
			return "", buildErr
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			buildErr = fmt.Errorf("failed to copy built plugin: %w", err)
			return "", buildErr
		}
	}

	// Clean up source directory
	os.RemoveAll(tempDir)

	return pluginSo.Name(), nil
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
	slog.Debug("Loading plugin", "name", pluginCfg.Name, "path", pluginCfg.Path, "url", pluginCfg.URL)

	pluginPath := pluginCfg.Path
	isTempFile := false
	var tempFilePath string

	// If URL is specified, build plugin from git repository
	if pluginCfg.URL != "" {
		slog.Info("Building plugin from git repository", "name", pluginCfg.Name, "url", pluginCfg.URL)
		builtPath, err := buildFromGit(pluginCfg.URL)
		if err != nil {
			return fmt.Errorf("failed to build plugin %s from git: %w", pluginCfg.Name, err)
		}
		pluginPath = builtPath
		isTempFile = true
		tempFilePath = builtPath
	} else if strings.HasPrefix(pluginCfg.Path, "http://") || strings.HasPrefix(pluginCfg.Path, "https://") {
		// If path is a URL, download the plugin to a temporary file
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
		isTempFile = true
		tempFilePath = tempFile.Name()
	}

	p, err := plugin.Open(pluginPath)
	if err != nil {
		// Clean up temporary files if we created them
		if isTempFile {
			os.Remove(tempFilePath)
		}
		return fmt.Errorf("failed to open plugin %s: %w", pluginCfg.Path, err)
	}

	sym, err := p.Lookup("NewDownloader")
	if err != nil {
		// Clean up temporary files if we created them
		if isTempFile {
			os.Remove(tempFilePath)
		}
		return fmt.Errorf("plugin %s does not export NewDownloader function: %w", pluginCfg.Name, err)
	}

	newDownloaderFunc, ok := sym.(func(map[string]interface{}) (Downloader, error))
	if !ok {
		// Clean up temporary files if we created them
		if isTempFile {
			os.Remove(tempFilePath)
		}
		return fmt.Errorf("plugin %s NewDownloader function has incorrect signature", pluginCfg.Name)
	}

	downloader, err := newDownloaderFunc(pluginCfg.Config)
	if err != nil {
		// Clean up temporary files if we created them
		if isTempFile {
			os.Remove(tempFilePath)
		}
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
	for k, v := range pm.downloaders {
		result[k] = v
	}
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
