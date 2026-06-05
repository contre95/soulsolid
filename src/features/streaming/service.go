package streaming

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
)

// Service handles audio streaming from both the download folder and the library.
type Service struct {
	queue   QueueLocator
	library LibraryLocator
	cfg     *config.Manager
}

// NewService creates a new streaming service.
func NewService(queue QueueLocator, library LibraryLocator, cfg *config.Manager) *Service {
	return &Service{queue: queue, library: library, cfg: cfg}
}

// QueueTrackStream returns the validated file path and MIME type for a pending queue item.
func (s *Service) QueueTrackStream(itemID string) (string, string, error) {
	path, err := s.queue.GetPendingTrackPath(itemID)
	if err != nil {
		return "", "", fmt.Errorf("queue item not found: %w", err)
	}
	downloadPath := filepath.Clean(s.cfg.Get().DownloadPath)
	if !strings.HasPrefix(filepath.Clean(path), downloadPath+string(filepath.Separator)) &&
		filepath.Clean(path) != downloadPath {
		return "", "", fmt.Errorf("track path outside allowed directory")
	}
	return path, mimeTypeFor(path), nil
}

// LibraryTrackStream returns the validated file path and MIME type for a library track.
func (s *Service) LibraryTrackStream(ctx context.Context, trackID string) (string, string, error) {
	path, err := s.library.GetLibraryTrackPath(ctx, trackID)
	if err != nil {
		return "", "", fmt.Errorf("track not found: %w", err)
	}
	libraryPath := filepath.Clean(s.cfg.Get().LibraryPath)
	if !strings.HasPrefix(filepath.Clean(path), libraryPath+string(filepath.Separator)) &&
		filepath.Clean(path) != libraryPath {
		return "", "", fmt.Errorf("track path outside allowed directory")
	}
	return path, mimeTypeFor(path), nil
}

func mimeTypeFor(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".flac") {
		return "audio/flac"
	}
	return "audio/mpeg"
}
