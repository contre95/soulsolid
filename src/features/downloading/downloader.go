package downloading

import (
	"fmt"

	"github.com/contre95/soulsolid/src/music"
)

// LinkResult represents the result of a link search, which can be tracks, albums, or an artist
type LinkResult struct {
	Type   string        `json:"type"` // "track", "album", "playlist", "artist"
	Tracks []music.Track `json:"tracks,omitempty"`
	Albums []music.Album `json:"albums,omitempty"`
	Artist *music.Artist `json:"artist,omitempty"`
}

// Downloader defines the interface for music downloaders
type Downloader interface {
	// Search methods
	SearchAlbums(query string, limit int) ([]music.Album, error)
	SearchTracks(query string, limit int) ([]music.Track, error)
	SearchArtists(query string, limit int) ([]music.Artist, error)
	SearchLinks(query string, limit int) (*LinkResult, error)
	// Navigation methods
	GetAlbumTracks(albumID string) ([]music.Track, error)
	GetArtistAlbums(artistID string) ([]music.Album, error)
	GetChartTracks(limit int) ([]music.Track, error)
	// Download methods
	DownloadTrack(trackID string, downloadDir string, progressCallback func(downloaded, total int64)) (*music.Track, error)
	DownloadAlbum(albumID string, downloadDir string, progressCallback func(downloaded, total int64)) ([]*music.Track, error)
	DownloadArtist(artistID string, downloadDir string, progressCallback func(downloaded, total int64)) ([]*music.Track, error)
	DownloadLink(url string, downloadDir string, progressCallback func(downloaded, total int64)) ([]*music.Track, error)
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
	SupportsSearch       bool `json:"supports_search"`
	SupportsArtistSearch bool `json:"supports_artist_search"`
	SupportsDirectLinks  bool `json:"supports_direct_links"`
	SupportsChartTracks  bool `json:"supports_chart_tracks"`
}

// ErrMethodNotSupported is returned when a downloader does not support a requested method
var ErrMethodNotSupported = fmt.Errorf("method not supported by this downloader")
