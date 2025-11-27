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
	libraryService music.LibraryService
	jobService     music.JobService
	config         *config.Manager
}

// NewService creates a new analyze service
func NewService(taggingService music.MetadataService, lyricsService music.LyricsService, libraryService music.LibraryService, jobService music.JobService, config *config.Manager) *Service {
	return &Service{
		taggingService: taggingService,
		lyricsService:  lyricsService,
		libraryService: libraryService,
		jobService:     jobService,
		config:         config,
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
func (s *Service) StartLyricsAnalysis(ctx context.Context) (string, error) {
	slog.Info("Starting lyrics analysis job")
	jobID, err := s.jobService.StartJob("analyze_lyrics", "Analyze Lyrics for Library", map[string]any{})
	if err != nil {
		return "", fmt.Errorf("failed to start lyrics analysis job: %w", err)
	}
	slog.Info("Lyrics analysis job started", "jobID", jobID)
	return jobID, nil
}
