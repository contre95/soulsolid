package duplicates

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/infra/fingerprint"
	"github.com/contre95/soulsolid/src/music"
)

// TagWriter interface for writing tags
type TagWriter interface {
	WriteFileTags(ctx context.Context, path string, track *music.Track) error
}

// Service provides duplicates analysis functionality
type Service struct {
	tagWriter  TagWriter
	library    music.Library
	config     *config.Manager
	queue      music.Queue
	jobService music.JobService
	similarity *fingerprint.SimilarityService
}

// NewService creates a new duplicates service
func NewService(tagWriter TagWriter, library music.Library, config *config.Manager, queue music.Queue, jobService music.JobService, similarity *fingerprint.SimilarityService) *Service {
	return &Service{
		tagWriter:  tagWriter,
		library:    library,
		config:     config,
		queue:      queue,
		jobService: jobService,
		similarity: similarity,
	}
}

// AddQueueItem adds a track to the duplicates queue
func (s *Service) AddQueueItem(track *music.Track, qType music.QueueItemType, metadata map[string]string) error {
	if track.ID == "" {
		return fmt.Errorf("track ID cannot be empty")
	}
	item := music.QueueItem{
		ID:        track.ID,
		Type:      qType,
		Track:     track,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
	return s.queue.Add(item)
}

// GetQueueItems returns all items in the duplicates queue
func (s *Service) GetQueueItems() map[string]music.QueueItem {
	return s.queue.GetAll()
}

// QueueCount returns the number of duplicates queue items
func (s *Service) QueueCount() int {
	items := s.GetQueueItems()
	count := 0
	for _, item := range items {
		if item.Type == "duplicate_fp_exact" || item.Type == "duplicate_fp_fuzzy" {
			count++
		}
	}
	return count
}

// ClearQueue removes all items from the duplicates queue
func (s *Service) ClearQueue() error {
	return s.queue.Clear()
}

// GetGroupedByFP returns queue items grouped by fingerprint group_key
func (s *Service) GetGroupedByFP() map[string][]music.QueueItem {
	items := s.GetQueueItems()
	groups := make(map[string][]music.QueueItem)
	for _, item := range items {
		if fp, ok := item.Metadata["group_fp"]; ok {
			groups[fp] = append(groups[fp], item)
		}
	}
	return groups
}

// ProcessQueueItem processes a duplicates queue item
func (s *Service) ProcessQueueItem(ctx context.Context, itemID string, action string) error {
	item, err := s.queue.GetByID(itemID)
	if err != nil {
		return fmt.Errorf("queue item not found: %w", err)
	}
	if item.Track == nil {
		return fmt.Errorf("queue item does not contain a valid track")
	}
	track := item.Track
	switch item.Type {
	case "duplicate_fp_exact", "duplicate_fp_fuzzy":
		switch action {
		case "delete":
			if err := os.Remove(track.Path); err != nil {
				slog.Warn("Failed to delete duplicate file", "path", track.Path, "error", err)
			}
			if err := s.library.DeleteTrack(ctx, track.ID); err != nil {
				return fmt.Errorf("failed to delete track: %w", err)
			}
			slog.Info("Deleted duplicate track", "trackID", track.ID)
		case "keep":
			slog.Info("Keeping duplicate track", "trackID", track.ID)
		case "merge_to_primary":
			primaryID := item.Metadata["primary_id"]
			if primaryID == "" {
				return fmt.Errorf("no primary_id in metadata")
			}
			primary, err := s.library.GetTrack(ctx, primaryID)
			if err != nil {
				return fmt.Errorf("failed to get primary track: %w", err)
			}
			primary.Metadata = track.Metadata
			primary.ModifiedDate = time.Now()
			if err := s.tagWriter.WriteFileTags(ctx, primary.Path, primary); err != nil {
				slog.Warn("Failed to write merged tags to primary", "error", err)
			}
			if err := s.library.UpdateTrack(ctx, primary); err != nil {
				return fmt.Errorf("failed to update primary track: %w", err)
			}
			if err := os.Remove(track.Path); err != nil {
				slog.Warn("Failed to delete dupe file after merge", "error", err)
			}
			if err := s.library.DeleteTrack(ctx, track.ID); err != nil {
				slog.Warn("Failed to delete dupe track after merge", "error", err)
			}
			slog.Info("Merged dupe to primary", "dupeID", track.ID, "primaryID", primaryID)
		default:
			return fmt.Errorf("invalid action '%s' for duplicate_fp", action)
		}
	default:
		return fmt.Errorf("unsupported queue item type: %s", item.Type)
	}
	return s.queue.Remove(itemID)
}

// ProcessQueueGroup processes all items in an FP group
func (s *Service) ProcessQueueGroup(ctx context.Context, groupFP string, action string) error {
	groups := s.GetGroupedByFP()
	groupItems, ok := groups[groupFP]
	if !ok || len(groupItems) == 0 {
		return fmt.Errorf("no items in group %s", groupFP)
	}
	for _, item := range groupItems {
		if err := s.ProcessQueueItem(ctx, item.ID, action); err != nil {
			slog.Warn("Failed to process group item", "itemID", item.ID, "groupFP", groupFP, "action", action, "error", err)
		}
	}
	return nil
}

// StartDuplicatesAnalysis starts the duplicates analysis job (always enabled, quick scan defaults)
func (s *Service) StartDuplicatesAnalysis(ctx context.Context, params map[string]any) (string, error) {
	if params == nil {
		params = map[string]any{
			"fp_exact_thresh": 0.95, // Quick Scan default
			"fp_fuzzy_thresh": 0.75,
			"default_action":  "queue",
		}
	}
	// Ensure required params (hardcoded rest)
	if _, ok := params["fp_exact_thresh"]; !ok {
		params["fp_exact_thresh"] = 0.95
	}
	if _, ok := params["fp_fuzzy_thresh"]; !ok {
		params["fp_fuzzy_thresh"] = 0.75
	}
	if _, ok := params["default_action"]; !ok {
		params["default_action"] = "queue"
	}
	jobID, err := s.jobService.StartJob("analyze_duplicates", "Analyze Library Duplicates (FP)", params)
	if err != nil {
		return "", fmt.Errorf("failed to start job: %w", err)
	}
	slog.Info("Duplicates analysis job started", "jobID", jobID, "params", params)
	return jobID, nil
}

// GetQueueItem returns a queue item by ID for comparison views
func (s *Service) GetQueueItem(id string) (music.QueueItem, error) {
	return s.queue.GetByID(id)
}
