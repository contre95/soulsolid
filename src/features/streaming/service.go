package streaming

import (
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

// Service handles audio streaming by validating and serving file paths.
type Service struct {
	cfg *config.Manager
}

// NewService creates a new streaming service.
func NewService(cfg *config.Manager) *Service {
	return &Service{cfg: cfg}
}

// Stream validates that path is within the library or download directory and
// returns the resolved path and MIME type.
func (s *Service) Stream(path string) (string, string, error) {
	cfg := s.cfg.Get()
	for _, base := range []string{cfg.LibraryPath, cfg.DownloadPath} {
		resolved, err := containedIn(path, base)
		if err == nil {
			return resolved, mimeTypeFor(resolved), nil
		}
	}
	return "", "", fmt.Errorf("track path outside allowed directories")
}

func mimeTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}
