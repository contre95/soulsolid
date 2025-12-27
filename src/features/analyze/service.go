package analyze

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service provides analysis functionality for batch operations on the library
type Service struct {
	taggingService music.MetadataService
	lyricsService  music.LyricsService
	library        music.Library
	jobService     music.JobService
	config         *config.Manager
	fileManager    music.FileManager
}

// NewService creates a new analyze service
func NewService(taggingService music.MetadataService, lyricsService music.LyricsService, library music.Library, jobService music.JobService, config *config.Manager, fileManager music.FileManager) *Service {
	return &Service{
		taggingService: taggingService,
		lyricsService:  lyricsService,
		library:        library,
		jobService:     jobService,
		config:         config,
		fileManager:    fileManager,
	}
}

// StartAcoustIDAnalysis starts a job to analyze all tracks for AcoustID
func (s *Service) StartAcoustIDAnalysis(ctx context.Context) (string, error) {
	slog.Info("Starting AcoustID analysis job")
	jobID, err := s.jobService.StartJob("analyze_acoustid", "Analyze AcoustID for Library", map[string]any{})
	if err != nil {
		return "", fmt.Errorf("failed to start AcoustID analysis job: %w", err)
	}
	slog.Info("AcoustID analysis job started", "jobID", jobID)
	return jobID, nil
}

// StartLyricsAnalysis starts a job to analyze all tracks for lyrics
func (s *Service) StartLyricsAnalysis(ctx context.Context, provider string) (string, error) {
	slog.Info("Starting lyrics analysis job", "provider", provider)
	jobID, err := s.jobService.StartJob("analyze_lyrics", "Analyze Lyrics for Library", map[string]any{
		"provider": provider,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start lyrics analysis job: %w", err)
	}
	slog.Info("Lyrics analysis job started", "jobID", jobID, "provider", provider)
	return jobID, nil
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
