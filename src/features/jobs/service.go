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
	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

type Job struct {
	ID         string
	Type       string
	Name       string
	Status     JobStatus
	Progress   int
	Message    string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Metadata   map[string]any
	cancelFunc context.CancelFunc
	Logger     *slog.Logger
	LogPath    string
	cancelled  bool // Track if job has been cancelled
}

type JobProgress struct {
	JobID    string
	Progress int
	Message  string
}

type TaskHandler interface {
	Execute(ctx context.Context, job *Job, progressChan chan<- JobProgress) error
	Cancel(jobID string) error
}

// Task defines the specific logic for a job type.
type Task interface {
	MetadataKeys() []string
	Execute(ctx context.Context, job *Job, progressUpdater func(int, string)) (map[string]any, error)
	Cleanup(job *Job) error
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
func (h *BaseTaskHandler) Execute(ctx context.Context, job *Job, progressChan chan<- JobProgress) error {
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
		progressChan <- JobProgress{
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

// JobService defines the interface for job management that other services will use
type JobService interface {
	StartJob(jobType string, name string, metadata map[string]any) (string, error)
	UpdateJobProgress(jobID string, progress int, message string)
	GetJob(jobID string) (*Job, bool)
	CancelJob(jobID string) error
	GetJobs() []*Job
}

type Service struct {
	jobs     map[string]*Job
	handlers map[string]TaskHandler
	mu       sync.RWMutex
	config   *config.Jobs
}

func NewService(cfg *config.Jobs) *Service {
	return &Service{
		jobs:     make(map[string]*Job),
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
	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Name:      name,
		Status:    JobStatusPending,
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
		job.Logger = slog.New(slog.NewTextHandler(logFile, nil))
		job.LogPath = logPath
	} else {
		// If logging is disabled, use a discard logger to prevent nil pointer errors
		job.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	s.mu.Lock()
	s.jobs[job.ID] = job

	// Check if we can start this job immediately
	if !s.isJobTypeRunning(jobType) {
		job.Status = JobStatusRunning
		s.mu.Unlock()
		go s.executeJob(job)
	} else {
		s.mu.Unlock()
	}

	return job.ID, nil
}

func (s *Service) executeJob(job *Job) {
	handler, exists := s.handlers[job.Type]
	if !exists {
		s.updateJobStatus(job.ID, JobStatusFailed, "No handler registered")
		return
	}
	progressChan := make(chan JobProgress, 10)
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	job.cancelFunc = cancel
	s.mu.Unlock()
	s.updateJobStatus(job.ID, JobStatusRunning, "Starting...")
	// Goroutine to listen for progress updates
	go func() {
		for progress := range progressChan {
			s.UpdateJobProgress(progress.JobID, progress.Progress, progress.Message)
		}
	}()
	err := handler.Execute(ctx, job, progressChan)
	close(progressChan)

	s.mu.Lock()
	cancelled := job.cancelled
	s.mu.Unlock()

	if err != nil {
		if errors.Is(err, context.Canceled) || cancelled {
			s.updateJobStatus(job.ID, JobStatusCancelled, "Job cancelled")
			s.executeWebhook(job)
		} else {
			// Check if this is a partial import success
			errMsg := err.Error()
			if strings.Contains(errMsg, "partial import:") && strings.Contains(errMsg, "tracks imported") {
				s.updateJobStatus(job.ID, JobStatusCompleted, "Job completed with errors - "+errMsg)
				s.executeWebhook(job)
			} else {
				s.updateJobStatus(job.ID, JobStatusFailed, err.Error())
				s.executeWebhook(job)
			}
		}
	} else {
		if cancelled {
			s.updateJobStatus(job.ID, JobStatusCancelled, "Job cancelled")
			s.executeWebhook(job)
		} else {
			s.updateJobStatus(job.ID, JobStatusCompleted, "Job completed successfully")
			s.executeWebhook(job)
		}
	}
	// After job completes, check for pending jobs of the same type
	s.startNextPendingJob(job.Type)
}

func (s *Service) updateJobStatus(jobID string, status JobStatus, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, exists := s.jobs[jobID]; exists {
		job.Status = status
		job.Message = message
		job.UpdatedAt = time.Now()
		if status == JobStatusCompleted {
			job.Progress = 100
		}
	}
}

func (s *Service) UpdateJobProgress(jobID string, progress int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, exists := s.jobs[jobID]; exists {
		// Don't update progress if job is in a terminal state
		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
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
	job.cancelled = true
	job.Status = JobStatusCancelled
	job.Message = "Job cancelled"
	job.UpdatedAt = time.Now()

	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	if handler, exists := s.handlers[job.Type]; exists {
		return handler.Cancel(jobID)
	}
	return nil
}

func (s *Service) GetJob(jobID string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	return job, exists
}

func (s *Service) GetJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Service) isJobTypeRunning(jobType string) bool {
	for _, job := range s.jobs {
		if job.Type == jobType && job.Status == JobStatusRunning {
			return true
		}
	}
	return false
}

func (s *Service) startNextPendingJob(jobType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Find the oldest pending job of this type
	var nextJob *Job
	for _, job := range s.jobs {
		if job.Type == jobType && job.Status == JobStatusPending {
			if nextJob == nil || job.CreatedAt.Before(nextJob.CreatedAt) {
				nextJob = job
			}
		}
	}
	if nextJob != nil {
		nextJob.Status = JobStatusRunning
		go s.executeJob(nextJob)
	}
}

func (s *Service) CleanupOldJobs(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, job := range s.jobs {
		if now.Sub(job.UpdatedAt) > maxAge &&
			(job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled) {
			if job.LogPath != "" {
				os.Remove(job.LogPath)
			}
			delete(s.jobs, id)
		}
	}
}

// executeWebhook executes the configured webhook command for job completion
func (s *Service) executeWebhook(job *Job) {
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
func (s *Service) executeWebhookCommand(command string, job *Job) {
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
