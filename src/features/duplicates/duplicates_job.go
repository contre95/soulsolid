package duplicates

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
)

// DuplicatesJobTask handles duplicates analysis job execution.
type DuplicatesJobTask struct {
	service *Service
}

// NewDuplicatesJobTask creates a new duplicates analysis job task.
func NewDuplicatesJobTask(service *Service) *DuplicatesJobTask {
	return &DuplicatesJobTask{service: service}
}

// MetadataKeys returns the required metadata keys for this job type.
func (t *DuplicatesJobTask) MetadataKeys() []string {
	return []string{}
}

// fpTrack holds the data needed for pairwise comparison without keeping full Track objects.
type fpTrack struct {
	track *music.Track
	fp    string
}

// Execute performs the duplicates analysis.
//
// Strategy:
//  1. Load all fingerprinted tracks from the library.
//  2. Compare every pair using Hamming distance on the raw chromaprint fingerprints.
//     Pairs whose duration difference exceeds 10 s are skipped as a fast pre-filter.
//  3. Union-find groups transitively similar tracks together.
//  4. For each group with ≥ 2 members the track with the earliest AddedDate is
//     designated as the primary; all others are queued for user review.
func (t *DuplicatesJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	job.Logger.Info("Duplicates analysis started", "color", "purple")

	exactThresh := 0.95
	fuzzyThresh := 0.75
	if v, ok := job.Metadata["fp_exact_thresh"].(float64); ok {
		exactThresh = v
	}
	if v, ok := job.Metadata["fp_fuzzy_thresh"].(float64); ok {
		fuzzyThresh = v
	}

	// ── Phase 1: load all fingerprinted tracks ────────────────────────────────
	totalTracks, err := t.service.library.GetTracksCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks count: %w", err)
	}
	if totalTracks == 0 {
		return map[string]any{"totalTracks": 0, "processed": 0, "queued": 0}, nil
	}

	progressUpdater(0, fmt.Sprintf("Loading %d tracks…", totalTracks))

	var candidates []fpTrack
	skipped := 0
	batchSize := 100
	for offset := 0; offset < totalTracks; offset += batchSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		batch, err := t.service.library.GetTracksPaginated(ctx, batchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get tracks (offset %d): %w", offset, err)
		}
		for _, tr := range batch {
			if tr.ChromaprintFingerprint != "" {
				candidates = append(candidates, fpTrack{track: tr, fp: tr.ChromaprintFingerprint})
			} else {
				slog.Debug("Skipping track: no fingerprint", "trackID", tr.ID, "title", tr.Title)
				skipped++
			}
		}
		pct := min(39, (offset*40)/totalTracks)
		progressUpdater(pct, fmt.Sprintf("Loaded %d / %d tracks", offset+len(batch), totalTracks))
	}

	n := len(candidates)
	if skipped > 0 {
		job.Logger.Warn(fmt.Sprintf("%d tracks skipped (no Chromaprint fingerprint) — these may have been imported before fingerprinting was supported", skipped), "color", "yellow")
	}
	job.Logger.Info("Fingerprinted tracks loaded", "count", n, "color", "blue")

	if n < 2 {
		progressUpdater(100, "No fingerprinted tracks to compare")
		return map[string]any{"totalTracks": totalTracks, "processed": n, "queued": 0}, nil
	}

	useAcoustID := false
	if v, ok := job.Metadata["use_acoustid"].(bool); ok {
		useAcoustID = v
	}

	// ── Phase 2: pairwise comparison with union-find ──────────────────────────
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(x, y int) {
		if px, py := find(x), find(y); px != py {
			parent[px] = py
		}
	}

	// pairSim stores the highest similarity seen for each non-primary member
	// (indexed by candidate index) so we can report it in queue metadata.
	pairSim := make([]float64, n)

	// ── Phase 2a: AcoustID pre-pass (optional) ────────────────────────────────
	// Group tracks that share an identical, non-empty AcoustID recording ID.
	// These are treated as exact matches regardless of Hamming distance.
	if useAcoustID {
		acoustidIndex := make(map[string][]int) // acoustid → candidate indices
		for i, c := range candidates {
			if aid, ok := c.track.Attributes["acoustid"]; ok && aid != "" {
				acoustidIndex[aid] = append(acoustidIndex[aid], i)
			}
		}
		acoustidGrouped := 0
		for _, idxs := range acoustidIndex {
			if len(idxs) < 2 {
				continue
			}
			for k := 1; k < len(idxs); k++ {
				union(idxs[0], idxs[k])
				pairSim[idxs[k]] = 1.0
				acoustidGrouped++
			}
		}
		if acoustidGrouped > 0 {
			job.Logger.Info(fmt.Sprintf("AcoustID pre-pass: %d additional duplicate pairs grouped", acoustidGrouped), "color", "blue")
		}
	}

	totalPairs := n * (n - 1) / 2
	pairsChecked := 0

	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		for j := i + 1; j < n; j++ {
			durI := candidates[i].track.Metadata.Duration
			durJ := candidates[j].track.Metadata.Duration
			if abs(durI-durJ) > 10 {
				pairsChecked++
				continue
			}

			sim := t.service.similarity.Hamming(candidates[i].fp, candidates[j].fp)
			if sim >= fuzzyThresh {
				union(i, j)
				if sim > pairSim[i] {
					pairSim[i] = sim
				}
				if sim > pairSim[j] {
					pairSim[j] = sim
				}
			}

			pairsChecked++
		}

		pct := 40 + (i*60)/n
		if pairsChecked%500 == 0 || i == n-1 {
			progressUpdater(pct, fmt.Sprintf("Comparing tracks… %d / %d pairs checked", pairsChecked, totalPairs))
		}
	}

	// ── Phase 3: build groups and queue non-primary tracks ────────────────────
	groups := make(map[int][]int)
	for i := range candidates {
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	queued := 0
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}

		// Primary = earliest AddedDate (first imported = most likely the original).
		primaryIdx := members[0]
		for _, idx := range members[1:] {
			if candidates[idx].track.AddedDate.Before(candidates[primaryIdx].track.AddedDate) {
				primaryIdx = idx
			}
		}
		primary := candidates[primaryIdx].track

		primaryArtist := "Unknown Artist"
		if len(primary.Artists) > 0 && primary.Artists[0].Artist != nil {
			primaryArtist = primary.Artists[0].Artist.Name
		}

		for _, idx := range members {
			if idx == primaryIdx {
				continue
			}
			track := candidates[idx].track
			sim := t.service.similarity.Hamming(candidates[idx].fp, candidates[primaryIdx].fp)

			qType := DuplicateFPFuzzy
			matchType := "fuzzy"
			if sim >= exactThresh {
				qType = DuplicateFPExact
				matchType = "exact"
			}

			md := map[string]string{
				"group_key":      primary.ID,
				"primary_id":     primary.ID,
				"primary_path":   primary.Path,
				"primary_title":  primary.Title,
				"primary_artist": primaryArtist,
				"similarity":     fmt.Sprintf("%.1f", sim*100),
				"match_type":     matchType,
			}
			if err := t.service.AddQueueItem(track, qType, md); err != nil {
				job.Logger.Warn("Failed to queue duplicate", "trackID", track.ID, "error", err)
			} else {
				queued++
			}
		}
	}

	msg := fmt.Sprintf("Done — %d tracks compared, %d duplicates queued, %d skipped (no fingerprint)", n, queued, skipped)
	job.Logger.Info(msg, "color", "green")
	progressUpdater(100, msg)

	return map[string]any{
		"totalTracks": totalTracks,
		"processed":   n,
		"queued":      queued,
		"skipped":     skipped,
	}, nil
}

// Cleanup is a no-op for this job.
func (t *DuplicatesJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up duplicates analysis job", "jobID", job.ID)
	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
