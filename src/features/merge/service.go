package merge

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/contre95/soulsolid/src/music"
)

// Library is the subset of the library repository the merge feature needs. It is satisfied by
// *database.SqliteLibrary; defined here so the feature depends only on what it uses.
type Library interface {
	GetArtists(ctx context.Context) ([]*music.Artist, error)
	GetAlbums(ctx context.Context) ([]*music.Album, error)
	GetGenres(ctx context.Context) ([]string, error)
	GetTrack(ctx context.Context, id string) (*music.Track, error)
	GetTracksFilteredPaginated(ctx context.Context, limit, offset int, filter *music.TrackFilter) ([]*music.Track, error)
	MergeArtists(ctx context.Context, canonicalID string, mergedIDs []string) error
	MergeAlbums(ctx context.Context, canonicalID string, mergedIDs []string) error
	StandardizeGenre(ctx context.Context, canonical string, variants []string) ([]string, error)
	GetDuplicateAcoustIDs(ctx context.Context, includeKept bool) (map[string][]string, error)
	KeepDuplicates(ctx context.Context, acoustID string, trackIDs []string) error
	MergeTracks(ctx context.Context, canonicalID string, mergedIDs []string) error
}

// FileDeleter removes a track file from disk (subset of music.FileManager).
type FileDeleter interface {
	DeleteTrack(ctx context.Context, trackPath string) error
}

// TagReader reads tags from a music file (subset of the shared infra implementation).
type TagReader interface {
	ReadFileTags(ctx context.Context, filePath string) (*music.Track, error)
}

// TagWriter writes tags into a music file (subset of the shared infra implementation).
type TagWriter interface {
	WriteFileTags(ctx context.Context, filePath string, track *music.Track) error
}

// Service detects and applies metadata merges (artists, albums, genres) and duplicate-track
// merges (same AcoustID; the losing files are deleted).
type Service struct {
	library     Library
	tagWriter   TagWriter
	tagReader   TagReader
	fileDeleter FileDeleter
	jobService  music.JobService
}

// NewService creates a new merge service.
func NewService(lib Library, tagWriter TagWriter, tagReader TagReader, fileDeleter FileDeleter, jobService music.JobService) *Service {
	return &Service{
		library:     lib,
		tagWriter:   tagWriter,
		tagReader:   tagReader,
		fileDeleter: fileDeleter,
		jobService:  jobService,
	}
}

// FindArtistGroups returns groups of artists whose names normalize to the same key.
func (s *Service) FindArtistGroups(ctx context.Context) ([]Group, error) {
	artists, err := s.library.GetArtists(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, a := range artists {
		if a == nil || a.Name == music.VariousArtistsName {
			continue
		}
		key := normalizeKey(a.Name)
		if key == "" {
			continue
		}
		buckets[key] = append(buckets[key], Variant{ID: a.ID, Value: a.Name})
	}
	return buildGroups(buckets), nil
}

// FindAlbumGroups returns groups of albums that share a normalized title AND primary artist.
func (s *Service) FindAlbumGroups(ctx context.Context) ([]Group, error) {
	albums, err := s.library.GetAlbums(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, al := range albums {
		if al == nil {
			continue
		}
		titleKey := normalizeKey(al.Title)
		if titleKey == "" {
			continue
		}
		artistName := ""
		if len(al.Artists) > 0 && al.Artists[0].Artist != nil {
			artistName = al.Artists[0].Artist.Name
		}
		key := titleKey + "\x00" + normalizeKey(artistName)
		buckets[key] = append(buckets[key], Variant{ID: al.ID, Value: al.Title, Sub: artistName})
	}
	return buildGroups(buckets), nil
}

// FindGenreGroups returns groups of genre strings that normalize to the same key.
func (s *Service) FindGenreGroups(ctx context.Context) ([]Group, error) {
	genres, err := s.library.GetGenres(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, g := range genres {
		key := normalizeKey(g)
		if key == "" {
			continue
		}
		buckets[key] = append(buckets[key], Variant{ID: g, Value: g})
	}
	return buildGroups(buckets), nil
}

// FindTrackGroups returns groups of tracks that share an AcoustID — i.e. the same recording in
// different files (formats/encodings). The suggested canonical is the highest-quality file.
// Groups the user already resolved with "Keep" are excluded, unless includeKept is set — which
// brings back every group regardless, so an accidental "Keep" can be found and corrected.
func (s *Service) FindTrackGroups(ctx context.Context, includeKept bool) ([]Group, error) {
	duplicates, err := s.library.GetDuplicateAcoustIDs(ctx, includeKept)
	if err != nil {
		return nil, err
	}
	groups := make([]Group, 0)
	for acoustID, trackIDs := range duplicates {
		tracks := make([]*music.Track, 0, len(trackIDs))
		for _, id := range trackIDs {
			t, err := s.library.GetTrack(ctx, id)
			if err != nil || t == nil {
				slog.Warn("skipping unresolvable duplicate track", "trackID", id, "acoustID", acoustID, "error", err)
				continue
			}
			tracks = append(tracks, t)
		}
		if len(tracks) < 2 {
			continue
		}
		variants := make([]Variant, len(tracks))
		for i, t := range tracks {
			variants[i] = Variant{ID: t.ID, Value: t.Title, Sub: t.Path}
		}
		sort.Slice(variants, func(i, j int) bool {
			if variants[i].Value != variants[j].Value {
				return variants[i].Value < variants[j].Value
			}
			return variants[i].Sub < variants[j].Sub
		})
		groups = append(groups, Group{Key: acoustID, Canonical: variants[0].ID, Variants: variants})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })
	return groups, nil
}

// KeepTracks records that the given tracks are intentional duplicates (e.g. a FLAC and an MP3 of
// the same recording kept on purpose). Only the database is touched — no files, no tags — and the
// group stops appearing in future duplicate scans.
func (s *Service) KeepTracks(ctx context.Context, acoustID string, trackIDs []string) error {
	if acoustID == "" {
		return fmt.Errorf("missing AcoustID group key")
	}
	if len(trackIDs) < 2 {
		return fmt.Errorf("a duplicate group needs at least two tracks")
	}
	return s.library.KeepDuplicates(ctx, acoustID, trackIDs)
}

// buildGroups keeps only buckets with two or more variants and attaches a smart canonical default.
func buildGroups(buckets map[string][]Variant) []Group {
	groups := make([]Group, 0)
	for key, variants := range buckets {
		if len(variants) < 2 {
			continue
		}
		sort.Slice(variants, func(i, j int) bool { return variants[i].Value < variants[j].Value })
		values := make([]string, len(variants))
		for i, v := range variants {
			values[i] = v.Value
		}
		groups = append(groups, Group{Key: key, Canonical: smartCanonical(values), Variants: variants})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })
	return groups
}

// StartMerge validates the selection and launches a background job that performs the merge in the
// database and rewrites the affected files' tags. For artists/albums canonical and members are
// entity IDs; for genres they are the raw genre strings.
func (s *Service) StartMerge(ctx context.Context, kind Kind, canonical string, members []string) (string, error) {
	switch kind {
	case KindArtist, KindAlbum, KindGenre:
	default:
		return "", fmt.Errorf("unknown merge kind %q", kind)
	}
	if canonical == "" {
		return "", fmt.Errorf("no canonical value selected")
	}
	if len(members) < 2 {
		return "", fmt.Errorf("a merge needs at least two members")
	}

	// Everything except the canonical one gets merged into it. The canonical must be selected.
	merged := make([]string, 0, len(members)-1)
	canonicalSelected := false
	for _, m := range members {
		if m == canonical {
			canonicalSelected = true
			continue
		}
		merged = append(merged, m)
	}
	if !canonicalSelected {
		return "", fmt.Errorf("the canonical value must be one of the selected items")
	}
	if len(merged) == 0 {
		return "", fmt.Errorf("nothing to merge into the canonical value")
	}

	jobID, err := music.StartTypedJob(s.jobService, "analyze_merge", fmt.Sprintf("Merge %ss", kind), MergeParams{
		Kind:      string(kind),
		Canonical: canonical,
		Merged:    merged,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start merge job: %w", err)
	}
	slog.Info("merge job started", "kind", kind, "canonical", canonical, "merged", merged, "jobID", jobID)
	return jobID, nil
}

// StartTrackMerge validates the selection and launches the background job that removes duplicate
// files. removeIDs are deleted from the library and from disk; the rest of allIDs stay, and
// playlist entries of the removed tracks are repointed to the highest-quality remaining track.
func (s *Service) StartTrackMerge(ctx context.Context, allIDs, removeIDs []string) (string, error) {
	if len(removeIDs) == 0 {
		return "", fmt.Errorf("check at least one file to merge away")
	}
	removeSet := make(map[string]bool, len(removeIDs))
	for _, id := range removeIDs {
		removeSet[id] = true
	}
	var kept []*music.Track
	for _, id := range allIDs {
		if id == "" || removeSet[id] {
			continue
		}
		t, err := s.library.GetTrack(ctx, id)
		if err != nil {
			return "", fmt.Errorf("failed to load track %s: %w", id, err)
		}
		if t != nil {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return "", fmt.Errorf("at least one file must be left unchecked to survive the merge")
	}
	canonical := kept[0].ID

	jobID, err := music.StartTypedJob(s.jobService, "analyze_merge", "Merge tracks", MergeParams{
		Kind:      string(KindTrack),
		Canonical: canonical,
		Merged:    removeIDs,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start merge job: %w", err)
	}
	slog.Info("track merge job started", "canonical", canonical, "merged", removeIDs, "jobID", jobID)
	return jobID, nil
}

// applyMerge performs the database merge and rewrites file tags (or, for tracks, deletes the
// duplicate files). It is invoked from the job task.
func (s *Service) applyMerge(ctx context.Context, job *music.Job, kind Kind, canonical string, merged []string, progress func(int, string)) (map[string]any, error) {
	if kind == KindTrack {
		return s.applyTrackMerge(ctx, job, canonical, merged, progress)
	}
	var affectedIDs []string
	switch kind {
	case KindArtist:
		if err := s.library.MergeArtists(ctx, canonical, merged); err != nil {
			return nil, fmt.Errorf("failed to merge artists: %w", err)
		}
		ids, err := s.affectedTrackIDs(ctx, &music.TrackFilter{ArtistIDs: []string{canonical}})
		if err != nil {
			return nil, err
		}
		affectedIDs = ids
	case KindAlbum:
		if err := s.library.MergeAlbums(ctx, canonical, merged); err != nil {
			return nil, fmt.Errorf("failed to merge albums: %w", err)
		}
		ids, err := s.affectedTrackIDs(ctx, &music.TrackFilter{AlbumIDs: []string{canonical}})
		if err != nil {
			return nil, err
		}
		affectedIDs = ids
	case KindGenre:
		ids, err := s.library.StandardizeGenre(ctx, canonical, merged)
		if err != nil {
			return nil, fmt.Errorf("failed to standardize genre: %w", err)
		}
		affectedIDs = ids
	default:
		return nil, fmt.Errorf("unknown merge kind %q", kind)
	}

	updated, failed := 0, 0
	total := len(affectedIDs)
	for i, id := range affectedIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if total > 0 {
			progress((i*100)/total, fmt.Sprintf("Rewriting file tags %d/%d", i+1, total))
		}
		if err := s.rewriteTrackTags(ctx, kind, id); err != nil {
			job.Logger.Warn("failed to rewrite file tags", "trackID", id, "error", err, "color", "orange")
			failed++
			continue
		}
		updated++
	}

	msg := fmt.Sprintf("Merged %d %s(s) into 1; rewrote %d file(s), %d failed.", len(merged), kind, updated, failed)
	job.Logger.Info("merge completed", "kind", kind, "merged", len(merged), "filesUpdated", updated, "filesFailed", failed, "color", "green")
	progress(100, msg)
	return map[string]any{
		"kind":         string(kind),
		"mergedCount":  len(merged),
		"filesUpdated": updated,
		"filesFailed":  failed,
		"msg":          msg,
	}, nil
}

// applyTrackMerge removes duplicate tracks (same AcoustID) from the database — repointing any
// playlist entries onto the canonical track — and hard-deletes their files from disk. The merged
// tracks are loaded up front because their paths are gone from the database afterwards.
func (s *Service) applyTrackMerge(ctx context.Context, job *music.Job, canonical string, merged []string, progress func(int, string)) (map[string]any, error) {
	canonicalTrack, err := s.library.GetTrack(ctx, canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to load canonical track: %w", err)
	}
	if canonicalTrack == nil {
		return nil, fmt.Errorf("canonical track not found: %s", canonical)
	}

	duplicates := make([]*music.Track, 0, len(merged))
	for _, id := range merged {
		t, err := s.library.GetTrack(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load duplicate track %s: %w", id, err)
		}
		if t == nil {
			job.Logger.Warn("duplicate track no longer exists, skipping", "trackID", id, "color", "orange")
			continue
		}
		duplicates = append(duplicates, t)
	}
	if len(duplicates) == 0 {
		return nil, fmt.Errorf("no duplicate tracks left to merge")
	}

	ids := make([]string, len(duplicates))
	for i, t := range duplicates {
		ids[i] = t.ID
	}
	if err := s.library.MergeTracks(ctx, canonical, ids); err != nil {
		return nil, fmt.Errorf("failed to merge tracks: %w", err)
	}

	deleted, failed := 0, 0
	total := len(duplicates)
	for i, t := range duplicates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		progress((i*100)/total, fmt.Sprintf("Deleting file %d/%d", i+1, total))
		if err := s.fileDeleter.DeleteTrack(ctx, t.Path); err != nil {
			job.Logger.Warn("failed to delete duplicate file", "path", t.Path, "error", err, "color", "orange")
			failed++
			continue
		}
		job.Logger.Info("deleted duplicate file", "path", t.Path)
		deleted++
	}

	msg := fmt.Sprintf("Kept %q; removed %d duplicate track(s), deleted %d file(s), %d failed.", canonicalTrack.Title, len(duplicates), deleted, failed)
	job.Logger.Info("track merge completed", "canonical", canonicalTrack.Path, "merged", len(duplicates), "filesDeleted", deleted, "filesFailed", failed, "color", "green")
	progress(100, msg)
	return map[string]any{
		"kind":         string(KindTrack),
		"mergedCount":  len(duplicates),
		"filesDeleted": deleted,
		"filesFailed":  failed,
		"msg":          msg,
	}, nil
}

// affectedTrackIDs returns the IDs of every track matching the filter, paginating to bound memory.
func (s *Service) affectedTrackIDs(ctx context.Context, filter *music.TrackFilter) ([]string, error) {
	var ids []string
	const batch = 200
	for offset := 0; ; offset += batch {
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, batch, offset, filter)
		if err != nil {
			return nil, err
		}
		for _, t := range tracks {
			ids = append(ids, t.ID)
		}
		if len(tracks) < batch {
			break
		}
	}
	return ids, nil
}

// rewriteTrackTags reads the track's current file tags, overrides only the merged dimension with
// the (now canonical) database values, and writes the tags back — preserving all other file data.
func (s *Service) rewriteTrackTags(ctx context.Context, kind Kind, trackID string) error {
	dbTrack, err := s.library.GetTrack(ctx, trackID)
	if err != nil {
		return err
	}
	if dbTrack == nil {
		return fmt.Errorf("track not found: %s", trackID)
	}
	fileTrack, err := s.tagReader.ReadFileTags(ctx, dbTrack.Path)
	if err != nil {
		return fmt.Errorf("read tags: %w", err)
	}
	fileTrack.ID = dbTrack.ID
	fileTrack.Path = dbTrack.Path
	switch kind {
	case KindArtist:
		fileTrack.Artists = dbTrack.Artists
		if dbTrack.Album != nil {
			fileTrack.Album = dbTrack.Album
		}
	case KindAlbum:
		fileTrack.Album = dbTrack.Album
	case KindGenre:
		fileTrack.Metadata.Genre = dbTrack.Metadata.Genre
	}
	return s.tagWriter.WriteFileTags(ctx, fileTrack.Path, fileTrack)
}
