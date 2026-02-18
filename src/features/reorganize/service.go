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

// StartReorganizeAnalysis starts a job to reorganize all tracks based on current path configuration
func (s *Service) StartReorganizeAnalysis(ctx context.Context) (string, error) {
	slog.Info("Starting file reorganization job")
	jobID, err := s.jobService.StartJob("analyze_reorganize", "Reorganize Library Files", map[string]any{})
	if err != nil {
		return "", fmt.Errorf("failed to start reorganization job: %w", err)
	}
	slog.Info("File reorganization job started", "jobID", jobID)
	return jobID, nil
}
