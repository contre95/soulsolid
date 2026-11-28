package albumcompleteness

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/music"
)

func TestComputeStatus(t *testing.T) {
	tests := []struct {
		name            string
		local, expected int
		wantStatus      Status
		wantMissing     int
	}{
		{"unknown without expected", 5, 0, StatusUnknown, 0},
		{"unknown with negative expected", 5, -1, StatusUnknown, 0},
		{"complete", 12, 12, StatusComplete, 0},
		{"extra tracks still complete", 14, 12, StatusComplete, 0},
		{"incomplete", 9, 12, StatusIncomplete, 3},
		{"empty album", 0, 10, StatusIncomplete, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, missing := computeStatus(tt.local, tt.expected)
			if status != tt.wantStatus || missing != tt.wantMissing {
				t.Errorf("computeStatus(%d, %d) = (%s, %d), want (%s, %d)", tt.local, tt.expected, status, missing, tt.wantStatus, tt.wantMissing)
			}
		})
	}
}

func TestFromAttributes(t *testing.T) {
	c := fromAttributes("a1", map[string]string{
		music.AlbumAttrExpectedTracks:        "12",
		music.AlbumAttrCompletenessSource:    "deezer",
		music.AlbumAttrCompletenessSourceURL: "https://example.test/a1",
		music.AlbumAttrCompletenessCheckedAt: "2026-01-02T03:04:05Z",
	}, 10)
	if c.Status != StatusIncomplete || c.Missing != 2 || c.ExpectedCount != 12 || c.LocalCount != 10 {
		t.Errorf("unexpected completeness: %+v", c)
	}
	if c.Manual || c.Source != "deezer" || c.SourceURL != "https://example.test/a1" || c.CheckedAt == "" {
		t.Errorf("unexpected source info: %+v", c)
	}

	manual := fromAttributes("a1", map[string]string{
		music.AlbumAttrExpectedTracks:     "8",
		music.AlbumAttrCompletenessSource: music.CompletenessSourceManual,
	}, 8)
	if !manual.Manual || manual.Status != StatusComplete {
		t.Errorf("unexpected manual completeness: %+v", manual)
	}

	if unknown := fromAttributes("a1", nil, 3); unknown.Status != StatusUnknown || unknown.ExpectedCount != 0 {
		t.Errorf("unexpected completeness without attributes: %+v", unknown)
	}
	if garbage := fromAttributes("a1", map[string]string{music.AlbumAttrExpectedTracks: "lots"}, 3); garbage.Status != StatusUnknown {
		t.Errorf("a non numeric count must be unknown, got %+v", garbage)
	}
}

func TestSearchParams(t *testing.T) {
	album := &music.Album{
		Title:          "Kid A",
		Barcode:        " 0724353 ",
		ReleaseGroupID: " rg ",
		Artists: []music.ArtistRole{
			{Artist: nil},
			{Artist: &music.Artist{Name: "  "}},
			{Artist: &music.Artist{Name: "Radiohead"}},
			{Artist: &music.Artist{Name: "Someone Else"}},
		},
	}
	got := searchParams(album)
	want := metadata.AlbumSearchParams{Album: "Kid A", AlbumArtist: "Radiohead", Barcode: "0724353", ReleaseGroupID: "rg"}
	if got != want {
		t.Errorf("searchParams() = %+v, want %+v", got, want)
	}
}

// fakeRepo is an in-memory LibraryRepo.
type fakeRepo struct {
	albums map[string]*music.Album
	local  map[string]int
	setErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{albums: map[string]*music.Album{}, local: map[string]int{}}
}

func (r *fakeRepo) addAlbum(id, title string, local int, attrs map[string]string) {
	if attrs == nil {
		attrs = map[string]string{}
	}
	r.albums[id] = &music.Album{
		ID: id, Title: title, Attributes: attrs,
		Artists: []music.ArtistRole{{Artist: &music.Artist{Name: "Artist"}}},
	}
	r.local[id] = local
}

func (r *fakeRepo) GetAlbum(_ context.Context, id string) (*music.Album, error) {
	return r.albums[id], nil
}

func (r *fakeRepo) GetAlbumSummaries(_ context.Context, limit, offset int, _ *music.AlbumFilter) ([]*music.AlbumSummary, error) {
	var all []*music.AlbumSummary
	for _, id := range r.sortedIDs() {
		all = append(all, &music.AlbumSummary{ID: id, Title: r.albums[id].Title, TrackCount: r.local[id]})
	}
	if offset >= len(all) {
		return nil, nil
	}
	return all[offset:min(offset+limit, len(all))], nil
}

func (r *fakeRepo) GetAlbumSummariesCount(context.Context, *music.AlbumFilter) (int, error) {
	return len(r.albums), nil
}

func (r *fakeRepo) GetTracksFilteredCount(_ context.Context, f *music.TrackFilter) (int, error) {
	return r.local[f.AlbumIDs[0]], nil
}

func (r *fakeRepo) SetAlbumAttributes(_ context.Context, id string, attrs map[string]string) error {
	if r.setErr != nil {
		return r.setErr
	}
	for k, v := range attrs {
		r.albums[id].Attributes[k] = v
	}
	return nil
}

func (r *fakeRepo) GetIncompleteAlbums(context.Context, int, int) ([]*music.AlbumCompletenessRow, error) {
	return nil, nil
}

func (r *fakeRepo) GetIncompleteAlbumsCount(context.Context) (int, error) { return 0, nil }

func (r *fakeRepo) sortedIDs() []string {
	ids := make([]string, 0, len(r.albums))
	for id := range r.albums {
		ids = append(ids, id)
	}
	// Insertion sort keeps the test free of extra imports and the order stable.
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

// fakeMetadata answers lookups by album title.
type fakeMetadata struct {
	infos map[string]*metadata.AlbumInfo
	errs  map[string]error
	calls []string
}

func (m *fakeMetadata) FetchAlbumInfo(_ context.Context, p metadata.AlbumSearchParams) (*metadata.AlbumInfo, error) {
	m.calls = append(m.calls, p.Album)
	if err := m.errs[p.Album]; err != nil {
		return nil, err
	}
	return m.infos[p.Album], nil
}

func newTestService(repo *fakeRepo, md *fakeMetadata) *Service {
	s := NewService(repo, md, nil)
	s.throttle = 0
	return s
}

func TestCheckAlbumStoresExpectedCount(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "Kid A", 8, nil)
	md := &fakeMetadata{infos: map[string]*metadata.AlbumInfo{
		"Kid A": {TotalTracks: 10, Source: "deezer", SourceURL: "https://example.test/kid-a"},
	}}
	s := newTestService(repo, md)

	c, err := s.CheckAlbum(context.Background(), "a1")
	if err != nil {
		t.Fatalf("CheckAlbum() error = %v", err)
	}
	if c.Status != StatusIncomplete || c.Missing != 2 || c.ExpectedCount != 10 || c.Source != "deezer" {
		t.Errorf("unexpected result: %+v", c)
	}
	attrs := repo.albums["a1"].Attributes
	if attrs[music.AlbumAttrExpectedTracks] != "10" || attrs[music.AlbumAttrCompletenessSource] != "deezer" || attrs[music.AlbumAttrCompletenessCheckedAt] == "" {
		t.Errorf("attributes not stored: %v", attrs)
	}

	stored, err := s.GetAlbumCompleteness(context.Background(), "a1")
	if err != nil || stored.Status != StatusIncomplete || stored.Missing != 2 {
		t.Errorf("GetAlbumCompleteness() = %+v, %v", stored, err)
	}
}

func TestCheckAlbumNoMatchKeepsPreviousCount(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "Obscure", 3, map[string]string{music.AlbumAttrExpectedTracks: "9"})
	s := newTestService(repo, &fakeMetadata{})

	if _, err := s.CheckAlbum(context.Background(), "a1"); !errors.Is(err, ErrNoMatch) {
		t.Fatalf("CheckAlbum() error = %v, want ErrNoMatch", err)
	}
	attrs := repo.albums["a1"].Attributes
	if attrs[music.AlbumAttrExpectedTracks] != "9" {
		t.Errorf("a failed lookup must keep the previous count, got %v", attrs)
	}
	if attrs[music.AlbumAttrCompletenessCheckedAt] == "" {
		t.Error("a failed lookup must still record the attempt")
	}
}

func TestCheckAlbumErrors(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "Kid A", 3, nil)
	boom := errors.New("provider down")
	s := newTestService(repo, &fakeMetadata{errs: map[string]error{"Kid A": boom}})

	if _, err := s.CheckAlbum(context.Background(), "a1"); !errors.Is(err, boom) {
		t.Errorf("CheckAlbum() error = %v, want %v", err, boom)
	}
	if _, err := s.CheckAlbum(context.Background(), "missing"); !errors.Is(err, ErrAlbumNotFound) {
		t.Errorf("CheckAlbum(missing) error = %v, want ErrAlbumNotFound", err)
	}
	if _, err := s.GetAlbumCompleteness(context.Background(), "missing"); !errors.Is(err, ErrAlbumNotFound) {
		t.Errorf("GetAlbumCompleteness(missing) error = %v, want ErrAlbumNotFound", err)
	}
}

func TestSetExpectedTracks(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "Kid A", 10, nil)
	s := newTestService(repo, &fakeMetadata{})

	c, err := s.SetExpectedTracks(context.Background(), "a1", 10)
	if err != nil {
		t.Fatalf("SetExpectedTracks() error = %v", err)
	}
	if c.Status != StatusComplete || !c.Manual {
		t.Errorf("unexpected result: %+v", c)
	}
	for _, bad := range []int{0, -3, 1000} {
		if _, err := s.SetExpectedTracks(context.Background(), "a1", bad); err == nil {
			t.Errorf("SetExpectedTracks(%d) must fail", bad)
		}
	}
	if _, err := s.SetExpectedTracks(context.Background(), "missing", 5); !errors.Is(err, ErrAlbumNotFound) {
		t.Errorf("SetExpectedTracks(missing) error = %v, want ErrAlbumNotFound", err)
	}
}

func runJob(t *testing.T, s *Service, params CompletenessParams) map[string]any {
	t.Helper()
	job := &music.Job{ID: "j1", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	stats, err := NewCompletenessJobTask(s).Execute(context.Background(), job, params, func(int, string) {})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stats["msg"] == "" || stats["msg"] == nil {
		t.Error("Execute() must return a msg")
	}
	return stats
}

func TestJobChecksSkipsAndCounts(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "Incomplete", 5, nil)
	repo.addAlbum("a2", "Complete", 10, nil)
	repo.addAlbum("a3", "Unknown", 4, nil)
	repo.addAlbum("a4", "Broken", 4, nil)
	repo.addAlbum("a5", "Checked Before", 4, map[string]string{music.AlbumAttrCompletenessCheckedAt: "2026-01-01T00:00:00Z"})
	repo.addAlbum("a6", "Manual", 4, map[string]string{
		music.AlbumAttrExpectedTracks:     "4",
		music.AlbumAttrCompletenessSource: music.CompletenessSourceManual,
	})
	md := &fakeMetadata{
		infos: map[string]*metadata.AlbumInfo{
			"Incomplete":     {TotalTracks: 8, Source: "deezer"},
			"Complete":       {TotalTracks: 10, Source: "deezer"},
			"Checked Before": {TotalTracks: 6, Source: "deezer"},
			"Manual":         {TotalTracks: 12, Source: "deezer"},
		},
		errs: map[string]error{"Broken": errors.New("boom")},
	}
	s := newTestService(repo, md)

	stats := runJob(t, s, CompletenessParams{})
	want := map[string]int{"totalAlbums": 6, "attempted": 6, "checked": 2, "incomplete": 1, "notFound": 1, "skipped": 2, "errors": 1}
	for k, v := range want {
		if stats[k] != v {
			t.Errorf("stats[%q] = %v, want %d (all stats: %v)", k, stats[k], v, stats)
		}
	}
	if repo.albums["a6"].Attributes[music.AlbumAttrExpectedTracks] != "4" {
		t.Error("the job must not replace a manual count")
	}
	for _, called := range md.calls {
		if called == "Checked Before" || called == "Manual" {
			t.Errorf("album %q must be skipped without a provider call", called)
		}
	}

	md.calls = nil
	forced := runJob(t, s, CompletenessParams{Force: true})
	if forced["skipped"] != 0 {
		t.Errorf("a forced run must skip nothing, got %v", forced)
	}
	if repo.albums["a6"].Attributes[music.AlbumAttrExpectedTracks] != "12" {
		t.Error("a forced run must replace a manual count")
	}
}

func TestJobStopsWhenCancelled(t *testing.T) {
	repo := newFakeRepo()
	repo.addAlbum("a1", "One", 1, nil)
	repo.addAlbum("a2", "Two", 1, nil)
	s := newTestService(repo, &fakeMetadata{})
	s.throttle = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	job := &music.Job{ID: "j1", Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if _, err := NewCompletenessJobTask(s).Execute(ctx, job, CompletenessParams{}, func(int, string) {}); !errors.Is(err, context.Canceled) {
		t.Errorf("Execute() error = %v, want context.Canceled", err)
	}
}
