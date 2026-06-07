package streaming

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
)

// containedIn guards against path traversal attacks: it resolves symlinks on
// both paths before the prefix check, so neither ../.. sequences nor symlinks
// inside the allowed directory can be used to escape it and read arbitrary files.
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

var audioMIME = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	// Not supported in soulsolid yet
	".wav":  "audio/wav",
	".aac":  "audio/aac",
	".m4a":  "audio/mp4",
	".ogg":  "audio/ogg",
	".opus": "audio/ogg",
	".wma":  "audio/x-ms-wma",
}

// Stream validates that path is within the library or download directory,
// has an allowed audio extension, and returns the resolved path and MIME type.
func (s *Service) Stream(path string) (string, string, error) {
	cfg := s.cfg.Get()
	for _, base := range []string{cfg.LibraryPath, cfg.DownloadPath} {
		resolved, err := containedIn(path, base)
		if err == nil {
			mime, ok := audioMIME[strings.ToLower(filepath.Ext(resolved))]
			if !ok {
				return "", "", fmt.Errorf("unsupported file type")
			}
			return resolved, mime, nil
		}
	}
	return "", "", fmt.Errorf("track path outside allowed directories")
}
