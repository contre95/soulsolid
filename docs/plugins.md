# Plugin Development Guide

This guide explains how to create and distribute downloaders as plugins for Soulsolid using Go's native plugin system.

## Overview

Soulsolid supports pluggable downloaders that can be loaded from `.so` files at runtime. This allows developers to create their own downloaders in separate repositories and distribute them independently.

## Plugin Interface

Your plugin must implement the `Downloader` interface defined in `src/features/downloading/downloader.go`:

```go
type Downloader interface {
    // Search methods
    SearchAlbums(query string, limit int) ([]music.Album, error)
    SearchTracks(query string, limit int) ([]music.Track, error)
    // Navigation methods
    GetAlbumTracks(albumID string) ([]music.Track, error)
    GetChartTracks(limit int) ([]music.Track, error)
    // Download methods
    DownloadTrack(trackID string, progressCallback func(downloaded, total int64)) (*music.Track, error)
    DownloadAlbum(albumID string) (*music.Album, error)
    // User info
    GetUserInfo() *UserInfo
    GetStatus() DownloaderStatus
    Name() string
}
```

## Creating a Plugin

1. **Create a new Go module for your plugin:**

```bash
mkdir my-downloader-plugin
cd my-downloader-plugin
go mod init github.com/yourusername/my-downloader-plugin
```

2. **Add Soulsolid as a dependency:**

```bash
go get github.com/contre95/soulsolid/src/music
go get github.com/contre95/soulsolid/src/features/downloading
```

3. **Implement your downloader:**

```go
package main

import (
    "github.com/contre95/soulsolid/src/features/downloading"
    "github.com/contre95/soulsolid/src/music"
)

type MyDownloader struct {
    // Your configuration fields
    apiKey string
    baseURL string
}

// Implement the Downloader interface
func (d *MyDownloader) Name() string {
    return "MyDownloader"
}

func (d *MyDownloader) SearchAlbums(query string, limit int) ([]music.Album, error) {
    // Implement album search
}

func (d *MyDownloader) SearchTracks(query string, limit int) ([]music.Track, error) {
    // Implement track search
}

func (d *MyDownloader) GetAlbumTracks(albumID string) ([]music.Track, error) {
    // Implement getting album tracks
}

func (d *MyDownloader) GetChartTracks(limit int) ([]music.Track, error) {
    // Implement getting chart tracks
}

func (d *MyDownloader) DownloadTrack(trackID string, progressCallback func(downloaded, total int64)) (*music.Track, error) {
    // Implement track download
}

func (d *MyDownloader) DownloadAlbum(albumID string) (*music.Album, error) {
    // Implement album download
}

func (d *MyDownloader) GetUserInfo() *downloading.UserInfo {
    // Return user information
}

func (d *MyDownloader) GetStatus() downloading.DownloaderStatus {
    // Return downloader status
}

// Export the NewDownloader function (this is required!)
func NewDownloader(config map[string]interface{}) (downloading.Downloader, error) {
    apiKey, ok := config["api_key"].(string)
    if !ok {
        return nil, fmt.Errorf("api_key is required")
    }

    baseURL, ok := config["base_url"].(string)
    if !ok {
        baseURL = "https://api.example.com" // default
    }

    return &MyDownloader{
        apiKey:  apiKey,
        baseURL: baseURL,
    }, nil
}
```

4. **Build the plugin:**

```bash
go build -buildmode=plugin -o mydownloader.so .
```

## Configuration

Plugins receive their configuration through the `NewDownloader` function as a `map[string]interface{}`. The configuration comes from the Soulsolid config file.

Example config.yaml:

```yaml
downloaders:
  plugins:
    - name: "mydownloader"
      path: "/path/to/mydownloader.so"
      config:
        api_key: "your_api_key_here"
        base_url: "https://api.example.com"
        timeout: 30
```

## Distribution

1. **Build for the target platform:** Make sure to build the plugin for the same OS and architecture as the Soulsolid binary.

2. **Distribute the .so file:** Users can place the `.so` file anywhere accessible to Soulsolid and configure the path in their config.

3. **Version compatibility:** Plugins should be built against the same version of Soulsolid to ensure API compatibility.

## Best Practices

- **Error handling:** Return meaningful errors from all methods.
- **Logging:** Use the standard `log/slog` package for logging.
- **Configuration validation:** Validate required configuration in `NewDownloader`.
- **Progress callbacks:** Implement progress callbacks for downloads to provide user feedback.
- **Status reporting:** Return appropriate status information in `GetStatus()`.
- **Thread safety:** Ensure your downloader is safe for concurrent use.

## Example Plugin

There's a dummy plugin I have created to serve as an example. You can find it [here](https://github.com/contre95/soulsolid-dummy-plugin).

## Testing

Test your plugin by:
1. Building it with `go build -buildmode=plugin`
2. Adding it to your Soulsolid config
3. Restarting Soulsolid
4. Testing the downloader through the web interface

## Troubleshooting

- **Plugin not loading:** Check that the path is correct and the file is readable.
- **Symbol not found:** Ensure you export `NewDownloader` (capital N).
- **Configuration errors:** Check that required config keys are provided.
- **Version mismatches:** Rebuild the plugin when updating Soulsolid.
