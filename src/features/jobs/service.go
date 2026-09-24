package jobs

import (
	"context"
	"encoding/json"
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
	"syscall"
	"text/template"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Ensure Service implements music.JobService interface
var _ music.JobService = (*Service)(nil)

type TaskHandler interface {
	Execute(ctx context.Context, job *music.Job, progressChan chan<- music.JobProgress) (map[string]any, error)
	Cancel(jobID string) error
}

// Task defines the specific logic for a job type. P is the task's parameter
// struct, decoded and validated from job.Metadata before Execute is called, so
// tasks never assert on map[string]any themselves. Job types that take no input
// declare their own empty struct.
//
// Nothing in this interface is exported back to the feature: tasks satisfy it
// structurally, so a feature package never has to import this one.
type Task[P any] interface {
	Execute(ctx context.Context, job *music.Job, params P, progressUpdater func(int, string)) (map[string]any, error)
	Cleanup(job *music.Job) error
}

// paramsValidator is shared: validator.Validate is safe for concurrent use and
// caches struct reflection, so it must not be rebuilt per job.
var paramsValidator = validator.New()

// BaseTaskHandler adapts a Task[P] to the TaskHandler the service calls. It owns
// every cross-cutting concern so no task can forget one: decoding metadata into
// P, validating it, adapting the progress channel to a plain callback, and
// running Cleanup via defer.
type BaseTaskHandler[P any] struct {
	Task Task[P]
}

// NewBaseTaskHandler wraps a typed task into a non-generic TaskHandler. The type
// parameter is erased here, which is what lets the service keep a single
// map[string]TaskHandler across heterogeneous job types.
func NewBaseTaskHandler[P any](task Task[P]) TaskHandler {
	return &BaseTaskHandler[P]{Task: task}
}

// decodeParams converts job.Metadata into P and validates it.
//
// The conversion goes through JSON rather than direct assertion so that values
// surviving a JSON round-trip ([]any vs []string, float64 vs int) normalize to
// the struct's declared types instead of every task hand-rolling a type switch.
func decodeParams[P any](metadata map[string]any) (P, error) {
	var params P

	// A nil/empty map is legitimate for tasks whose fields are all optional;
	// validation below is what rejects it when fields are required.
	if len(metadata) > 0 {
		raw, err := json.Marshal(metadata)
		if err != nil {
			return params, fmt.Errorf("invalid job metadata: %w", err)
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			return params, fmt.Errorf("invalid job metadata: %w", err)
		}
	}

	if err := paramsValidator.Struct(params); err != nil {
		var invalid *validator.InvalidValidationError
		// Non-struct P (e.g. a type alias) can't be validated; that's not a job error.
		if errors.As(err, &invalid) {
			return params, nil
		}
		return params, fmt.Errorf("invalid job metadata: %w", err)
	}

	return params, nil
}

// Execute runs the job using the provided task and returns any result stats for
// the caller to merge into job.Metadata under the appropriate lock.
func (h *BaseTaskHandler[P]) Execute(ctx context.Context, job *music.Job, progressChan chan<- music.JobProgress) (map[string]any, error) {
	if job.Logger != nil {
		job.Logger.Info("Starting job", "name", job.Name)
	}

	params, err := decodeParams[P](job.Metadata)
	if err != nil {
		if job.Logger != nil {
			job.Logger.Error("Error: " + err.Error())
		}
		return nil, err
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

	stats, err := h.Task.Execute(ctx, job, params, progressUpdater)
	if err != nil {
		if job.Logger != nil {
			job.Logger.Error("Error during job execution", "error", err)
		}
		return stats, err
	}

	if job.Logger != nil {
		job.Logger.Info("Job finished successfully", "name", job.Name)
	}
	return stats, nil
}

// Cancel stops a running job.
// The actual cancellation is handled by the context in the job service,
// this method is for any specific cleanup required by the handler.
func (h *BaseTaskHandler[P]) Cancel(jobID string) error {
	// Specific cancellation logic can be implemented in the task if needed.
	return nil
}

type Service struct {
	jobs     map[string]*music.Job
	handlers map[string]TaskHandler
	mu       sync.RWMutex
	config   *config.Manager
}

func NewService(cfg *config.Manager) *Service {
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

	if s.config.Get().Jobs.Log {
		logDir := s.config.Get().Jobs.LogPath
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
	stats, err := handler.Execute(ctx, job, progressChan)
	close(progressChan)

	s.mu.Lock()
	cancelled := job.Cancelled
	if stats != nil {
		if job.Metadata == nil {
			job.Metadata = make(map[string]any)
		}
		maps.Copy(job.Metadata, stats)
	}
	s.mu.Unlock()

	switch {
	case errors.Is(err, context.Canceled) || cancelled:
		s.updateJobStatus(job.ID, music.JobStatusCancelled, "Job cancelled")
	case errors.Is(err, music.ErrJobPartialSuccess):
		s.updateJobStatus(job.ID, music.JobStatusCompleted, "Job completed with errors - "+err.Error())
	case err != nil:
		s.updateJobStatus(job.ID, music.JobStatusFailed, err.Error())
	default:
		s.updateJobStatus(job.ID, music.JobStatusCompleted, "Job completed successfully")
	}
	// Read the job back as a snapshot so the webhook doesn't touch shared state.
	if snap, ok := s.GetJob(job.ID); ok {
		s.executeWebhook(snap)
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

// SetJobName renames a job (e.g. once a download task learns the real title).
// Tasks must use this instead of writing job.Name directly, so the write happens
// under the service lock and cannot race with handlers serializing the job.
func (s *Service) SetJobName(jobID string, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, exists := s.jobs[jobID]; exists {
		job.Name = name
		job.UpdatedAt = time.Now()
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

// snapshotJob returns a copy of job (with its own Metadata map) that is safe to
// read and JSON-serialize without holding s.mu. Handlers must never receive the
// live *music.Job: running tasks and the service mutate it concurrently.
func snapshotJob(job *music.Job) *music.Job {
	j := *job
	j.Metadata = maps.Clone(job.Metadata)
	return &j
}

func (s *Service) GetJob(jobID string) (*music.Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, false
	}
	return snapshotJob(job), true
}

func (s *Service) GetJobs() []*music.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*music.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, snapshotJob(job))
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
	if !s.config.Get().Jobs.Webhooks.Enabled {
		return
	}

	// Check if this job type should trigger webhooks
	shouldNotify := false
	for _, jobType := range s.config.Get().Jobs.Webhooks.JobTypes {
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
	tmpl, err := template.New("webhook").Parse(s.config.Get().Jobs.Webhooks.Command)
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
	// Put the shell and all its children in a new process group so the timeout
	// kill reaches the entire tree and no grandchildren are orphaned to PID 1.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	timer := time.AfterFunc(30*time.Second, func() {
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
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
