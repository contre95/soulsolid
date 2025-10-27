package downloading

import "github.com/contre95/soulsolid/src/music"

// Downloader defines the interface for music downloaders
type Downloader interface {
	// Search methods
	SearchAlbums(query string, limit int) ([]music.Album, error)
	SearchTracks(query string, limit int) ([]music.Track, error)
	// Navigation methods
	GetAlbumTracks(albumID string) ([]music.Track, error)
	GetChartTracks(limit int) ([]music.Track, error)
	// Download methods
	DownloadTrack(trackID string, downloadDir string, progressCallback func(downloaded, total int64)) (*music.Track, error)
	DownloadAlbum(albumID string, downloadDir string, progressCallback func(downloaded, total int64)) ([]*music.Track, error)
	// User info
	GetUserInfo() *UserInfo
	GetStatus() DownloaderStatus
	Name() string
	Capabilities() DownloaderCapabilities
}

// UserInfo represents user information from a downloader
type UserInfo struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Link         string `json:"link"`
	Picture      string `json:"picture"`
	PictureSmall string `json:"picture_small"`
	Country      string `json:"country"`
	Tracklist    string `json:"tracklist"`
	Type         string `json:"type"`
	UserOptions  any    `json:"user_options"`
}

// DownloaderCapabilities represents the capabilities of a downloader
type DownloaderCapabilities struct {
	SupportsSearch      bool `json:"supports_search"`
	SupportsDirectLinks bool `json:"supports_direct_links"`
}
