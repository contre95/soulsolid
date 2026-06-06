package streaming

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
)

// containedIn resolves symlinks on both paths and checks that candidate is
// inside (or equal to) base, preventing symlink escapes.
func containedIn(candidate, base string) (string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	resolvedBase, err := filepath.EvalSymlinks(filepath.Clean(base))
	if err != nil {
		return "", fmt.Errorf("cannot resolve base path: %w", err)
	}
	if resolved != resolvedBase && !strings.HasPrefix(resolved, resolvedBase+string(filepath.Separator)) {
		return "", fmt.Errorf("track path outside allowed directory")
	}
	return resolved, nil
}

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
	resolved, err := containedIn(path, s.cfg.Get().DownloadPath)
	if err != nil {
		return "", "", err
	}
	return resolved, mimeTypeFor(resolved), nil
}

// LibraryTrackStream returns the validated file path and MIME type for a library track.
func (s *Service) LibraryTrackStream(ctx context.Context, trackID string) (string, string, error) {
	path, err := s.library.GetLibraryTrackPath(ctx, trackID)
	if err != nil {
		return "", "", fmt.Errorf("track not found: %w", err)
	}
	resolved, err := containedIn(path, s.cfg.Get().LibraryPath)
	if err != nil {
		return "", "", err
	}
	return resolved, mimeTypeFor(resolved), nil
}

func mimeTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}
