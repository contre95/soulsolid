package dummy

import (
	"fmt"
	"time"

	"soulsolid/src/features/downloading"
	"soulsolid/src/music"
)

// DummyDownloader implements the Downloader interface with hardcoded data
type DummyDownloader struct{}

// NewDummyDownloader creates a new dummy downloader
func NewDummyDownloader() downloading.Downloader {
	return &DummyDownloader{}
}

// Name returns the name of this downloader
func (d *DummyDownloader) Name() string {
	return "Dummy"
}

// GetUserInfo returns hardcoded user information
func (d *DummyDownloader) GetUserInfo() *downloading.UserInfo {
	return &downloading.UserInfo{
		ID:           12345,
		Name:         "Demo User",
		Link:         "https://example.com/user/demo",
		Picture:      "/img/user_demo.jpg",
		PictureSmall: "/img/user_demo.jpg",
		Country:      "AR",
		Tracklist:    "https://example.com/user/demo/tracks",
		Type:         "user",
		UserOptions: map[string]any{
			"lossless":  true,
			"hq":        true,
			"streaming": true,
			"offline":   true,
		},
	}
}

// dummyArtist returns a hardcoded artist
func dummyArtist() *music.Artist {
	return &music.Artist{
		ID:          "dummy-artist-1",
		Name:        "Demo Artist",
		ImageSmall:  "/img/artist.svg",
		ImageMedium: "/img/artist.svg",
		ImageLarge:  "/img/artist.svg",
		ImageXL:     "/img/artist.svg",
	}
}

// dummyAlbum returns a hardcoded album
func dummyAlbum() *music.Album {
	return &music.Album{
		ID:          "dummy-album-1",
		Title:       "Demo Album",
		Type:        music.AlbumTypeDefault,
		Artists:     []music.ArtistRole{{Artist: dummyArtist(), Role: "main"}},
		ReleaseDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Genre:       "Pop",
		ImageSmall:  "/img/album_demo.jpg",
		ImageMedium: "/img/album_demo.jpg",
		ImageLarge:  "/img/album_demo.jpg",
		ImageXL:     "/img/album_demo.jpg",
	}
}

// dummyTrack returns a hardcoded track
func dummyTrack() *music.Track {
	return &music.Track{
		ID:         "dummy-track-1",
		Title:      "Demo Track",
		Artists:    []music.ArtistRole{{Artist: dummyArtist(), Role: "main"}},
		Album:      dummyAlbum(),
		ISRC:       "US1234567890",
		PreviewURL: "https://example.com/preview.mp3",
		Metadata: music.Metadata{
			Duration:       180,
			TrackNumber:    1,
			DiscNumber:     1,
			Year:           2023,
			Genre:          "Pop",
			OriginalYear:   2023,
			ExplicitLyrics: false,
			BPM:            120.0,
		},
		Bitrate:         320,
		Format:          "mp3",
		SampleRate:      44100,
		BitDepth:        16,
		Channels:        2,
		ExplicitContent: false,
	}
}

// dummyAudioData returns some dummy audio data (a small MP3 header for demonstration)
func dummyAudioData() []byte {
	// This is a minimal MP3 frame header for demonstration
	// In reality, this would be actual audio data
	return []byte{
		0xFF, 0xFB, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Add more dummy data to make it look like audio
		0x49, 0x44, 0x33, 0x03, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x54, 0x49, 0x54, 0x32, 0x00, 0x00,
		0x00, 0x0A, 0x00, 0x00, 0x00, 0x44, 0x65, 0x6D,
		0x6F, 0x20, 0x54, 0x72, 0x61, 0x63, 0x6B,
	}
}

// SearchAlbums returns dummy album search results
func (d *DummyDownloader) SearchAlbums(query string, limit int) ([]music.Album, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	demoImages := []string{"/img/album_demo.jpg"}

	results := make([]music.Album, 0, limit)
	for i := 0; i < limit; i++ {
		album := *dummyAlbum()
		album.ID = fmt.Sprintf("dummy-album-%d", i+1)
		album.Title = fmt.Sprintf("Demo Album %d", i+1)
		imageIndex := i % len(demoImages)
		album.ImageSmall = demoImages[imageIndex]
		album.ImageMedium = demoImages[imageIndex]
		album.ImageLarge = demoImages[imageIndex]
		album.ImageXL = demoImages[imageIndex]
		results = append(results, album)
	}
	return results, nil
}

// SearchTracks returns dummy track search results
func (d *DummyDownloader) SearchTracks(query string, limit int) ([]music.Track, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	results := make([]music.Track, 0, limit)
	for i := 0; i < limit; i++ {
		track := *dummyTrack()
		track.ID = fmt.Sprintf("dummy-track-%d", i+1)
		track.Title = fmt.Sprintf("Demo Track %d", i+1)
		results = append(results, track)
	}
	return results, nil
}

// GetAlbumTracks returns dummy tracks for an album
func (d *DummyDownloader) GetAlbumTracks(albumID string) ([]music.Track, error) {
	tracks := make([]music.Track, 10)
	for i := 0; i < 10; i++ {
		track := *dummyTrack()
		track.ID = fmt.Sprintf("%s-track-%d", albumID, i+1)
		track.Title = fmt.Sprintf("Track %d", i+1)
		track.Metadata.TrackNumber = i + 1
		tracks[i] = track
	}
	return tracks, nil
}

// GetChartTracks returns dummy chart tracks
func (d *DummyDownloader) GetChartTracks(limit int) ([]music.Track, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	results := make([]music.Track, 0, limit)
	for i := 0; i < limit; i++ {
		track := *dummyTrack()
		track.ID = fmt.Sprintf("chart-track-%d", i+1)
		track.Title = fmt.Sprintf("Chart Track %d", i+1)
		results = append(results, track)
	}
	return results, nil
}

// DownloadTrack returns a dummy track with hardcoded audio data
func (d *DummyDownloader) DownloadTrack(trackID string, progressCallback func(downloaded, total int64)) (*music.Track, error) {
	track := dummyTrack()
	track.ID = trackID
	track.Data = dummyAudioData()

	// Simulate progress
	if progressCallback != nil {
		total := int64(len(track.Data))
		for downloaded := int64(0); downloaded <= total; downloaded += total / 10 {
			progressCallback(downloaded, total)
		}
	}

	return track, nil
}

// DownloadAlbum returns a dummy album with tracks
func (d *DummyDownloader) DownloadAlbum(albumID string) (*music.Album, error) {
	album := dummyAlbum()
	album.ID = albumID

	// Add tracks to the album
	tracks := make([]*music.Track, 10)
	for i := 0; i < 10; i++ {
		track := dummyTrack()
		track.ID = fmt.Sprintf("%s-track-%d", albumID, i+1)
		track.Title = fmt.Sprintf("Track %d", i+1)
		track.Metadata.TrackNumber = i + 1
		track.Data = dummyAudioData()
		tracks[i] = track
	}
	album.Tracks = tracks

	return album, nil
}

// GetStatus returns the current status of the dummy downloader
func (d *DummyDownloader) GetStatus() downloading.DownloaderStatus {
	return downloading.DownloaderStatus{
		Name:    "dummy",
		Status:  "valid",
		Message: "Demo mode active",
	}
}
