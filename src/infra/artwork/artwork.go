package artwork

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
	"github.com/go-flac/flacpicture"
	goflac "github.com/go-flac/go-flac"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
	_ "image/gif"
)

// Service handles artwork downloading, caching, and embedding
type Service struct {
	config *config.Manager
}

// NewService creates a new artwork service
func NewService(config *config.Manager) *Service {
	return &Service{
		config: config,
	}
}

// DownloadArtwork downloads artwork from URL to a temporary file with caching
func (s *Service) DownloadArtwork(ctx context.Context, url string) (string, func(), error) {
	if url == "" {
		return "", nil, fmt.Errorf("empty artwork URL")
	}

	// Create temp directory if it doesn't exist
	tempDir := "/tmp/soulsolid-artwork"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate cache key from URL
	hash := md5.Sum([]byte(url))
	cacheKey := fmt.Sprintf("%x", hash)
	ext := getImageExtension(url)
	tempPath := filepath.Join(tempDir, cacheKey+ext)

	// Check if file already exists and is recent (cache for 24 hours)
	if info, err := os.Stat(tempPath); err == nil {
		if time.Since(info.ModTime()) < 24*time.Hour {
			slog.Debug("Using cached artwork", "path", tempPath)
			return tempPath, func() { s.cleanupTempArtwork(tempPath) }, nil
		}
		// Remove old cache file
		os.Remove(tempPath)
	}

	// Download the image
	slog.Debug("Downloading artwork", "url", url, "path", tempPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", nil, fmt.Errorf("artwork download failed with status %d", resp.StatusCode)
	}

	// Create the file
	file, err := os.Create(tempPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Copy the response body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(tempPath) // Clean up on error
		return "", nil, fmt.Errorf("failed to write artwork file: %w", err)
	}

	slog.Debug("Artwork downloaded successfully", "path", tempPath)
	return tempPath, func() { s.cleanupTempArtwork(tempPath) }, nil
}

// GetArtworkForTrack gets the appropriate artwork path for a track
func (s *Service) GetArtworkForTrack(ctx context.Context, track *music.Track) (string, func(), error) {
	// Get artwork URL - prefer album cover, fallback to artist image
	var imageURL string
	if track.Album != nil && track.Album.ImageLarge != "" {
		imageURL = track.Album.ImageLarge
	} else if len(track.Artists) > 0 && track.Artists[0].Artist != nil && track.Artists[0].Artist.ImageLarge != "" {
		imageURL = track.Artists[0].Artist.ImageLarge
	}

	if imageURL == "" {
		return "", nil, nil // No artwork available
	}

	return s.DownloadArtwork(ctx, imageURL)
}

// SaveLocalArtwork saves artwork as a local file in the specified directory
func (s *Service) SaveLocalArtwork(ctx context.Context, track *music.Track, dirPath string) error {
	cfg := s.config.Get()

	// Check if local artwork is enabled
	if !cfg.Downloaders.Artwork.Local.Enabled {
		return nil
	}

	artworkPath, cleanup, err := s.GetArtworkForTrack(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to get artwork: %w", err)
	}
	if artworkPath == "" {
		slog.Debug("No artwork URL available for track", "trackID", track.ID)
		return nil
	}
	defer cleanup()

	// Read and decode image
	imgData, err := os.ReadFile(artworkPath)
	if err != nil {
		return fmt.Errorf("failed to read artwork file: %w", err)
	}

	img, _, err := image.Decode(strings.NewReader(string(imgData)))
	if err != nil {
		return fmt.Errorf("failed to decode artwork image: %w", err)
	}

	// Resize image if needed
	if cfg.Downloaders.Artwork.Local.Size > 0 {
		img = resize.Resize(uint(cfg.Downloaders.Artwork.Local.Size), uint(cfg.Downloaders.Artwork.Local.Size), img, resize.Lanczos3)
	}

	// Create output file path
	localPath := filepath.Join(dirPath, cfg.Downloaders.Artwork.Local.Template)
	if cfg.Downloaders.Artwork.Local.Template == "" {
		localPath = filepath.Join(dirPath, "cover.jpg")
	}

	// Create output file
	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create artwork file: %w", err)
	}
	defer outFile.Close()

	// Encode and save image
	options := &jpeg.Options{Quality: 85}
	if err := jpeg.Encode(outFile, img, options); err != nil {
		return fmt.Errorf("failed to encode artwork image: %w", err)
	}

	slog.Info("Saved local artwork", "path", localPath, "trackID", track.ID)
	return nil
}

// EmbedArtwork embeds artwork into audio file
func (s *Service) EmbedArtwork(audioPath, artworkPath string) error {
	if artworkPath == "" {
		slog.Debug("No artwork to embed")
		return nil
	}

	ext := strings.ToLower(filepath.Ext(audioPath))
	switch ext {
	case ".mp3":
		return s.embedArtworkInMP3(audioPath, artworkPath)
	case ".flac":
		return s.embedArtworkInFLAC(audioPath, artworkPath)
	default:
		slog.Warn("Unsupported audio format for artwork embedding", "format", ext)
		return nil
	}
}

// embedArtworkInMP3 embeds artwork into MP3 file using ID3 tags
func (s *Service) embedArtworkInMP3(audioPath, artworkPath string) error {
	// Read artwork file
	artworkData, err := os.ReadFile(artworkPath)
	if err != nil {
		return fmt.Errorf("failed to read artwork file: %w", err)
	}

	// Determine MIME type
	mimeType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(artworkPath), ".png") {
		mimeType = "image/png"
	}

	// Open MP3 file
	tag, err := id3v2.Open(audioPath, id3v2.Options{Parse: false})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file for tagging: %w", err)
	}
	defer tag.Close()

	// Resize image if configured
	cfg := s.config.Get()
	if cfg.Downloaders.Artwork.Embedded.Enabled && cfg.Downloaders.Artwork.Embedded.Size > 0 {
		resizedData, err := s.resizeImage(artworkData, cfg.Downloaders.Artwork.Embedded.Size)
		if err != nil {
			slog.Warn("Failed to resize artwork for MP3", "error", err)
		} else {
			artworkData = resizedData
		}
	}

	// Create APIC frame
	pic := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    mimeType,
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     artworkData,
	}

	// Add the picture frame
	tag.AddAttachedPicture(pic)

	// Save the tag
	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	slog.Debug("Artwork embedded in MP3 file", "path", audioPath)
	return nil
}

// embedArtworkInFLAC embeds artwork into FLAC file using PICTURE metadata block
func (s *Service) embedArtworkInFLAC(audioPath, artworkPath string) error {
	// Read artwork file
	artworkData, err := os.ReadFile(artworkPath)
	if err != nil {
		return fmt.Errorf("failed to read artwork file: %w", err)
	}

	// Resize image if configured
	cfg := s.config.Get()
	if cfg.Downloaders.Artwork.Embedded.Enabled && cfg.Downloaders.Artwork.Embedded.Size > 0 {
		resizedData, err := s.resizeImage(artworkData, cfg.Downloaders.Artwork.Embedded.Size)
		if err != nil {
			slog.Warn("Failed to resize artwork for FLAC", "error", err)
		} else {
			artworkData = resizedData
		}
	}

	// Detect MIME type from data
	mimeType := "image/jpeg" // default
	if len(artworkData) >= 4 {
		if string(artworkData[:4]) == "\x89PNG" {
			mimeType = "image/png"
		} else if string(artworkData[:2]) == "\xFF\xD8" {
			mimeType = "image/jpeg"
		}
	}

	// Parse the FLAC file
	f, err := goflac.ParseFile(audioPath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	// Create PICTURE metadata block using flacpicture library
	pic, _ := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Front cover", artworkData, mimeType)
	marshaled := pic.Marshal()
	pictureBlock := &goflac.MetaDataBlock{
		Type: goflac.Picture,
		Data: marshaled.Data,
	}
	f.Meta = append(f.Meta, pictureBlock)

	slog.Debug("Artwork embedded in FLAC file", "path", audioPath)
	return nil
}

// resizeImage resizes image data to the specified max size
func (s *Service) resizeImage(imgData []byte, maxSize int) ([]byte, error) {
	img, _, err := image.Decode(strings.NewReader(string(imgData)))
	if err != nil {
		return nil, err
	}

	resized := resize.Resize(uint(maxSize), uint(maxSize), img, resize.Lanczos3)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// cleanupTempArtwork removes temporary artwork files
func (s *Service) cleanupTempArtwork(paths ...string) {
	for _, path := range paths {
		if path != "" && strings.HasPrefix(path, "/tmp/soulsolid-artwork/") {
			if err := os.Remove(path); err != nil {
				slog.Warn("Failed to cleanup temp artwork file", "path", path, "error", err)
			} else {
				slog.Debug("Cleaned up temp artwork file", "path", path)
			}
		}
	}
}

// getImageExtension extracts file extension from URL or defaults to .jpg
func getImageExtension(url string) string {
	if strings.Contains(url, ".png") {
		return ".png"
	}
	if strings.Contains(url, ".jpg") || strings.Contains(url, ".jpeg") {
		return ".jpg"
	}
	return ".jpg" // default
}
