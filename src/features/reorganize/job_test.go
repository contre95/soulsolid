package reorganize

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/contre95/soulsolid/src/infra/files"
	"github.com/contre95/soulsolid/src/music"
)

// albumAndTitle renders a track's path from its album and title alone, so
// tracks sharing both share a path.
type albumAndTitle struct{}

func (albumAndTitle) RenderPath(track *music.Track) (string, error) {
	return filepath.Join(track.Album.Title, "00 "+track.Title), nil
}

// library holds tracks in order, enforces unique paths like the database, and
// fails to update the tracks in locked.
type library struct {
	music.Library
	tracks []*music.Track
	locked map[string]bool
}

func (l *library) GetTracksCount(context.Context) (int, error) { return len(l.tracks), nil }

func (l *library) GetTracksPaginated(_ context.Context, limit, offset int) ([]*music.Track, error) {
	page := []*music.Track{}
	for _, track := range l.tracks[offset:min(offset+limit, len(l.tracks))] {
		loaded := *track
		page = append(page, &loaded)
	}
	return page, nil
}

func (l *library) FindTrackByPath(_ context.Context, path string) (*music.Track, error) {
	for _, track := range l.tracks {
		if track.Path == path {
			return track, nil
		}
	}
	return nil, nil
}

func (l *library) UpdateTrack(_ context.Context, track *music.Track) error {
	if l.locked[track.ID] {
		return errors.New("database is locked")
	}
	for _, stored := range l.tracks {
		if stored.ID != track.ID && stored.Path == track.Path {
			return errors.New("UNIQUE constraint failed: tracks.path")
		}
	}
	for i, stored := range l.tracks {
		if stored.ID == track.ID {
			l.tracks[i] = track
		}
	}
	return nil
}

// path returns the stored path of the track with id.
func (l *library) path(id string) string {
	for _, track := range l.tracks {
		if track.ID == id {
			return track.Path
		}
	}
	return ""
}

// unsorted writes a track's file outside its library path, holding its id.
func unsorted(t *testing.T, root, id, album, title string) *music.Track {
	t.Helper()
	path := filepath.Join(root, "unsorted", id+".mp3")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(id), 0o644); err != nil {
		t.Fatal(err)
	}
	return &music.Track{ID: id, Path: path, Title: title, Album: &music.Album{Title: album}}
}

func reorganize(t *testing.T, root string, lib *library) map[string]any {
	t.Helper()
	organizer := files.NewFileOrganizer(func() string { return root }, func() string { return filepath.Join(root, "inbox") }, albumAndTitle{}, func() bool { return false })
	job := &music.Job{Metadata: map[string]any{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	result, err := NewReorganizeJobTask(NewService(lib, organizer, nil, nil, files.NewPathSanitizer())).Execute(context.Background(), job, ReorganizeParams{}, func(int, string) {})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// expectFile fails unless path holds content.
func expectFile(t *testing.T, path, content string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected %s to hold %q: %v", path, content, err)
	}
	if string(got) != content {
		t.Errorf("expected %s to hold %q, got %q", path, content, got)
	}
}

func expectCounts(t *testing.T, result map[string]any, moved, conflicts, errs int) {
	t.Helper()
	if result["moved"] != moved || result["conflicts"] != conflicts || result["errors"] != errs {
		t.Errorf("expected %d moved, %d conflicts, %d errors, got %v moved, %v conflicts, %v errors",
			moved, conflicts, errs, result["moved"], result["conflicts"], result["errors"])
	}
}

func TestReorganizeSkipsTracksWhosePathIsTaken(t *testing.T) {
	root := t.TempDir()
	lib := &library{tracks: []*music.Track{
		unsorted(t, root, "first", "Festival", "Live Set"),
		unsorted(t, root, "second", "Festival", "Live Set"),
		unsorted(t, root, "third", "Festival", "Live Set"),
	}}
	unsortedPaths := map[string]string{"second": lib.tracks[1].Path, "third": lib.tracks[2].Path}
	shared := filepath.Join(root, "Festival", "00 Live Set.mp3")

	expectCounts(t, reorganize(t, root, lib), 1, 2, 0)

	if got := lib.path("first"); got != shared {
		t.Errorf("expected the first track at %s, got %s", shared, got)
	}
	expectFile(t, shared, "first")
	for id, path := range unsortedPaths {
		if got := lib.path(id); got != path {
			t.Errorf("expected %s to stay at %s, got %s", id, path, got)
		}
		expectFile(t, path, id)
	}
}

func TestReorganizeSkipsAFileAlreadyAtThePath(t *testing.T) {
	root := t.TempDir()
	stray := filepath.Join(root, "Album", "00 Song.mp3")
	if err := os.MkdirAll(filepath.Dir(stray), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stray, []byte("stray"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := &library{tracks: []*music.Track{unsorted(t, root, "song", "Album", "Song")}}
	original := lib.tracks[0].Path

	expectCounts(t, reorganize(t, root, lib), 0, 1, 0)

	if got := lib.path("song"); got != original {
		t.Errorf("expected the track to stay at %s, got %s", original, got)
	}
	expectFile(t, original, "song")
	expectFile(t, stray, "stray")
}

func TestReorganizeSkipsAPathAnotherTrackHoldsWithoutItsFile(t *testing.T) {
	root := t.TempDir()
	held := filepath.Join(root, "Album", "00 Song.mp3")
	lib := &library{tracks: []*music.Track{
		unsorted(t, root, "song", "Album", "Song"),
		{ID: "missing", Path: held, Title: "Song", Album: &music.Album{Title: "Album"}},
	}}
	original := lib.tracks[0].Path

	expectCounts(t, reorganize(t, root, lib), 0, 1, 0)

	if got := lib.path("song"); got != original {
		t.Errorf("expected the track to stay at %s, got %s", original, got)
	}
	expectFile(t, original, "song")
	if _, err := os.Stat(held); !os.IsNotExist(err) {
		t.Errorf("expected nothing written to %s, got %v", held, err)
	}
}

func TestReorganizeMovesATrackBackWhenItsPathIsNotSaved(t *testing.T) {
	root := t.TempDir()
	lib := &library{tracks: []*music.Track{unsorted(t, root, "song", "Album", "Song")}, locked: map[string]bool{"song": true}}
	original := lib.tracks[0].Path

	expectCounts(t, reorganize(t, root, lib), 0, 0, 1)

	expectFile(t, original, "song")
	if _, err := os.Stat(filepath.Join(root, "Album")); !os.IsNotExist(err) {
		t.Errorf("expected the destination directory removed, got %v", err)
	}
}
