package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
)

// Ensure Service implements music.JobService interface
var _ music.JobService = (*Service)(nil)

type TaskHandler interface {
	Execute(ctx context.Context, job *music.Job, progressChan chan<- music.JobProgress) error
	Cancel(jobID string) error
}

// Task defines the specific logic for a job type.
type Task interface {
	MetadataKeys() []string
	Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error)
	Cleanup(job *music.Job) error
}

// BaseTaskHandler provides a base implementation for TaskHandler.
type BaseTaskHandler struct {
	Task Task
}

// NewBaseTaskHandler creates a new BaseTaskHandler.
func NewBaseTaskHandler(task Task) *BaseTaskHandler {
	return &BaseTaskHandler{Task: task}
}

// Execute runs the job using the provided task.
func (h *BaseTaskHandler) Execute(ctx context.Context, job *music.Job, progressChan chan<- music.JobProgress) error {
	if job.Logger != nil {
		job.Logger.Info("Starting job", "name", job.Name)
	}

	// Validate metadata
	for _, key := range h.Task.MetadataKeys() {
		if _, ok := job.Metadata[key]; !ok {
			err := fmt.Errorf("missing %s in job metadata", key)
			if job.Logger != nil {
				job.Logger.Error("Error: " + err.Error())
			}
			return err
		}
	}

	progressUpdater := func(percentage int, status string) {
		progressChan <- music.JobProgress{
			JobID:    job.ID,
			Progress: percentage,
			Message:  status,
		}
		if job.Logger != nil {
			job.Logger.Info("Progress", "percentage", percentage, "status", status)
		}
	}

	// Defer cleanup
	defer func() {
		if err := h.Task.Cleanup(job); err != nil {
			if job.Logger != nil {
				job.Logger.Error("Error during job cleanup", "error", err)
			}
		}
	}()

	stats, err := h.Task.Execute(ctx, job, progressUpdater)
	// Merge stats into job metadata even on error
	if stats != nil {
		if job.Metadata == nil {
			job.Metadata = make(map[string]any)
		}
		maps.Copy(job.Metadata, stats)
	}
	if err != nil {
		if job.Logger != nil {
			job.Logger.Error("Error during job execution", "error", err)
		}
		return err
	}

	if job.Logger != nil {
		job.Logger.Info("Job finished successfully", "name", job.Name)
	}
	return nil
}

// Cancel stops a running job.
// The actual cancellation is handled by the context in the job service,
// this method is for any specific cleanup required by the handler.
func (h *BaseTaskHandler) Cancel(jobID string) error {
	// Specific cancellation logic can be implemented in the task if needed.
	return nil
}

type Service struct {
	jobs     map[string]*music.Job
	handlers map[string]TaskHandler
	mu       sync.RWMutex
	config   *config.Jobs
}

func NewService(cfg *config.Jobs) *Service {
	return &Service{
		jobs:     make(map[string]*music.Job),
		handlers: make(map[string]TaskHandler),
		config:   cfg,
	}
}

func (s *Service) RegisterHandler(jobType string, handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[jobType] = handler
}

func (s *Service) StartJob(jobType string, name string, metadata map[string]any) (string, error) {
	// Create a copy of jobType to prevent potential memory sharing issues
	jobTypeCopy := strings.Clone(jobType)
	job := &music.Job{
		ID:        uuid.New().String(),
		Type:      jobTypeCopy,
		Name:      name,
		Status:    music.JobStatusPending,
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
	}

	if s.config.Log {
		logDir := s.config.LogPath
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create log directory: %w", err)
		}
		logName := fmt.Sprintf("%s-%s.log", time.Now().Format("2006-01-02"), job.ID)
		logPath := filepath.Join(logDir, logName)
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return "", fmt.Errorf("failed to open log file: %w", err)
		}
		job.Logger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo}))
		job.LogPath = logPath
	} else {
		// If logging is disabled, use a discard logger to prevent nil pointer errors
		job.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	s.mu.Lock()
	s.jobs[job.ID] = job

	// Check if we can start this job immediately
	if !s.isAnyJobRunning() {
		job.Status = music.JobStatusRunning
		s.mu.Unlock()
		go s.executeJob(job)
	} else {
		s.mu.Unlock()
	}

	return job.ID, nil
}

func (s *Service) executeJob(job *music.Job) {
	handler, exists := s.handlers[job.Type]
	if !exists {
		s.updateJobStatus(job.ID, music.JobStatusFailed, "No handler registered")
		return
	}
	progressChan := make(chan music.JobProgress, 10)
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	job.CancelFunc = cancel
	s.mu.Unlock()
	s.updateJobStatus(job.ID, music.JobStatusRunning, "Starting...")
	// Goroutine to listen for progress updates
	go func() {
		for progress := range progressChan {
			s.UpdateJobProgress(progress.JobID, progress.Progress, progress.Message)
		}
	}()
	err := handler.Execute(ctx, job, progressChan)
	close(progressChan)

	s.mu.Lock()
	cancelled := job.Cancelled
	s.mu.Unlock()

	if err != nil {
		if errors.Is(err, context.Canceled) || cancelled {
			s.updateJobStatus(job.ID, music.JobStatusCancelled, "Job cancelled")
			s.executeWebhook(job)
		} else {
			// Check if this is a partial import success
			errMsg := err.Error()
			if strings.Contains(errMsg, "partial import:") && strings.Contains(errMsg, "tracks imported") {
				s.updateJobStatus(job.ID, music.JobStatusCompleted, "Job completed with errors - "+errMsg)
				s.executeWebhook(job)
			} else {
				s.updateJobStatus(job.ID, music.JobStatusFailed, err.Error())
				s.executeWebhook(job)
			}
		}
	} else {
		if cancelled {
			s.updateJobStatus(job.ID, music.JobStatusCancelled, "Job cancelled")
			s.executeWebhook(job)
		} else {
			s.updateJobStatus(job.ID, music.JobStatusCompleted, "Job completed successfully")
			s.executeWebhook(job)
		}
	}
	// After job completes, check for pending jobs
	s.startNextPendingJob()
}

func (s *Service) updateJobStatus(jobID string, status music.JobStatus, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, exists := s.jobs[jobID]; exists {
		job.Status = status
		job.Message = message
		job.UpdatedAt = time.Now()
		if status == music.JobStatusCompleted {
			job.Progress = 100
		}
	}
}

func (s *Service) UpdateJobProgress(jobID string, progress int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, exists := s.jobs[jobID]; exists {
		// Don't update progress if job is in a terminal state
		if job.Status == music.JobStatusCompleted || job.Status == music.JobStatusFailed || job.Status == music.JobStatusCancelled {
			return
		}
		job.Progress = progress
		job.Message = message
		job.UpdatedAt = time.Now()
	}
}

func (s *Service) CancelJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}

	// Mark job as cancelled and update status
	job.Cancelled = true
	job.Status = music.JobStatusCancelled
	job.Message = "Job cancelled"
	job.UpdatedAt = time.Now()

	if job.CancelFunc != nil {
		job.CancelFunc()
	}
	if handler, exists := s.handlers[job.Type]; exists {
		return handler.Cancel(jobID)
	}
	return nil
}

func (s *Service) GetJob(jobID string) (*music.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	return job, exists
}

func (s *Service) GetJobs() []*music.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*music.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Service) ClearFinishedJobs() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, job := range s.jobs {
		if job.Status == music.JobStatusCompleted || job.Status == music.JobStatusFailed || job.Status == music.JobStatusCancelled {
			if job.LogPath != "" {
				os.Remove(job.LogPath)
			}
			delete(s.jobs, id)
		}
	}
	return nil
}

func (s *Service) isAnyJobRunning() bool {
	for _, job := range s.jobs {
		if job.Status == music.JobStatusRunning {
			return true
		}
	}
	return false
}

func (s *Service) startNextPendingJob() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Find the oldest pending job
	var nextJob *music.Job
	for _, job := range s.jobs {
		if job.Status == music.JobStatusPending {
			if nextJob == nil || job.CreatedAt.Before(nextJob.CreatedAt) {
				nextJob = job
			}
		}
	}
	if nextJob != nil {
		nextJob.Status = music.JobStatusRunning
		go s.executeJob(nextJob)
	}
}

func (s *Service) CleanupOldJobs(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, job := range s.jobs {
		if now.Sub(job.UpdatedAt) > maxAge &&
			(job.Status == music.JobStatusCompleted || job.Status == music.JobStatusFailed || job.Status == music.JobStatusCancelled) {
			if job.LogPath != "" {
				os.Remove(job.LogPath)
			}
			delete(s.jobs, id)
		}
	}
}

// executeWebhook executes the configured webhook command for job completion
func (s *Service) executeWebhook(job *music.Job) {
	if !s.config.Webhooks.Enabled {
		return
	}

	// Check if this job type should trigger webhooks
	shouldNotify := false
	for _, jobType := range s.config.Webhooks.JobTypes {
		if jobType == job.Type || jobType == "*" {
			shouldNotify = true
			break
		}
	}

	if !shouldNotify {
		return
	}

	// Prepare template data
	message := job.Message
	if job.Metadata != nil {
		if msg, ok := job.Metadata["msg"].(string); ok && msg != "" {
			message = msg
		}
	}

	data := struct {
		Name     string
		Type     string
		Status   string
		Message  string
		Duration string
	}{
		Name:     job.Name,
		Type:     job.Type,
		Status:   string(job.Status),
		Message:  message,
		Duration: time.Since(job.CreatedAt).Round(time.Second).String(),
	}

	// Execute template
	tmpl, err := template.New("webhook").Parse(s.config.Webhooks.Command)
	if err != nil {
		if job.Logger != nil {
			job.Logger.Error("Failed to parse webhook template", "error", err)
		}
		return
	}

	var command strings.Builder
	if err := tmpl.Execute(&command, data); err != nil {
		if job.Logger != nil {
			job.Logger.Error("Failed to execute webhook template", "error", err)
		}
		return
	}

	// Execute command asynchronously
	go func(cmd string) {
		s.executeWebhookCommand(cmd, job)
	}(command.String())
}

// executeWebhookCommand executes the webhook command safely
func (s *Service) executeWebhookCommand(command string, job *music.Job) {
	// Use shell to properly handle quoted strings and complex commands
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = os.Environ()

	// Set timeout
	timer := time.AfterFunc(30*time.Second, func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	if err := cmd.Run(); err != nil {
		if job.Logger != nil {
			job.Logger.Error("Webhook execution failed", "command", command, "error", err)
		}
	} else {
		if job.Logger != nil {
			job.Logger.Info("Webhook executed successfully", "command", command)
		}
	}
}
