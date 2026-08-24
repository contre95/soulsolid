package merge

import (
	"context"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
)

// MergeJobTask applies a metadata merge (DB + file tags) in the background.
type MergeJobTask struct {
	service *Service
}

// NewMergeJobTask creates a new merge job task.
func NewMergeJobTask(service *Service) *MergeJobTask {
	return &MergeJobTask{service: service}
}

// MergeParams is the metadata contract for a merge job.
type MergeParams struct {
	Kind      string   `json:"kind" validate:"required"`
	Canonical string   `json:"canonical" validate:"required"`
	Merged    []string `json:"merged" validate:"required,min=1,dive,required"`
}

// Execute applies the merge described by the job metadata.
func (t *MergeJobTask) Execute(ctx context.Context, job *music.Job, params MergeParams, progressUpdater func(int, string)) (map[string]any, error) {
	job.Logger.Info("starting merge", "kind", params.Kind, "canonical", params.Canonical, "merged", params.Merged, "color", "blue")
	progressUpdater(0, "Starting merge")
	return t.service.applyMerge(ctx, job, Kind(params.Kind), params.Canonical, params.Merged, progressUpdater)
}

// Cleanup is a no-op for merge jobs.
func (t *MergeJobTask) Cleanup(job *music.Job) error {
	slog.Debug("cleaning up merge job", "jobID", job.ID)
	return nil
}
