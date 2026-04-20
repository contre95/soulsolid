package reorganize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service is the domain service for the reorganize feature.
type Service struct {
	fileManager music.FileManager
	library     music.Library
	config      *config.Manager
	jobService  music.JobService
}

// NewService creates a new reorganize service.
func NewService(lib music.Library, fileManager music.FileManager, cfg *config.Manager, jobService music.JobService) *Service {
	return &Service{
		library:     lib,
		fileManager: fileManager,
		config:      cfg,
		jobService:  jobService,
	}
}

// StartReorganizeAnalysis starts a job to reorganize all tracks based on current path configuration.
// When fat32Safe is true the job will also strip FAT32-forbidden characters from every path segment.
func (s *Service) StartReorganizeAnalysis(ctx context.Context, fat32Safe bool) (string, error) {
	slog.Info("Starting file reorganization job", "fat32Safe", fat32Safe)
	jobID, err := s.jobService.StartJob("analyze_reorganize", "Reorganize Library Files", map[string]any{
		"fat32_safe": fat32Safe,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start reorganization job: %w", err)
	}
	slog.Info("File reorganization job started", "jobID", jobID)
	return jobID, nil
}
