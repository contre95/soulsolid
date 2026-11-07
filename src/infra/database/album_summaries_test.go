package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/contre95/soulsolid/src/music"
)

// seedAlbumLibrary builds a small library:
//   - "Zeta" by Alpha (2001): 3 tracks across 2 discs, genres Rock (2) and Electronic (1)
//   - "Beta" by Bravo (release date unknown, tracks from 1999): 1 track, genre Jazz
//   - "Empty" by Alpha: no tracks
func seedAlbumLibrary(t *testing.T) (*SqliteLibrary, context.Context) {
	t.Helper()
	ctx := context.Background()
	lib, err := NewSqliteLibrary(filepath.Join(t.TempDir(), "lib.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	alpha := &music.Artist{ID: "ar-alpha", Name: "Alpha"}
	bravo := &music.Artist{ID: "ar-bravo", Name: "Bravo"}
	for _, a := range []*music.Artist{alpha, bravo} {
		if err := lib.AddArtist(ctx, a); err != nil {
			t.Fatalf("add artist: %v", err)
		}
	}
	main := func(a *music.Artist) []music.ArtistRole { return []music.ArtistRole{{Artist: a, Role: "main"}} }

	zeta := &music.Album{ID: "al-zeta", Title: "Zeta", Artists: main(alpha), ReleaseDate: time.Date(2001, 5, 1, 0, 0, 0, 0, time.UTC)}
	beta := &music.Album{ID: "al-beta", Title: "Beta", Artists: main(bravo), ReleaseDate: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)}
	empty := &music.Album{ID: "al-empty", Title: "Empty", Artists: main(alpha)}
	for _, al := range []*music.Album{zeta, beta, empty} {
		if err := lib.AddAlbum(ctx, al); err != nil {
			t.Fatalf("add album: %v", err)
		}
	}

	day := func(d int) time.Time { return time.Date(2024, 1, d, 0, 0, 0, 0, time.UTC) }
	tracks := []*music.Track{
		// Inserted out of order on purpose: disc 2 first, then disc 1 track 2, then disc 1 track 1.
		{ID: "t-z-2-1", Path: "/m/z21", Title: "Z disc2 one", Artists: main(alpha), Album: zeta, AddedDate: day(1),
			Metadata: music.Metadata{DiscNumber: 2, TrackNumber: 1, Duration: 100, Genre: "Electronic", Year: 2001}},
		{ID: "t-z-1-2", Path: "/m/z12", Title: "Z disc1 two", Artists: main(alpha), Album: zeta, AddedDate: day(2),
			Metadata: music.Metadata{DiscNumber: 1, TrackNumber: 2, Duration: 200, Genre: "Rock", Year: 2001}},
		{ID: "t-z-1-1", Path: "/m/z11", Title: "Z disc1 one", Artists: main(alpha), Album: zeta, AddedDate: day(3),
			Metadata: music.Metadata{DiscNumber: 1, TrackNumber: 1, Duration: 300, Genre: "Rock", Year: 2001}},
		{ID: "t-b-1", Path: "/m/b1", Title: "B one", Artists: main(bravo), Album: beta, AddedDate: day(9),
			Metadata: music.Metadata{DiscNumber: 1, TrackNumber: 1, Duration: 50, Genre: "Jazz", Year: 1999}},
	}
	for _, tr := range tracks {
		if err := lib.AddTrack(ctx, tr); err != nil {
			t.Fatalf("add track %s: %v", tr.ID, err)
		}
	}
	return lib, ctx
}

func summaryIDs(s []*music.AlbumSummary) []string {
	ids := make([]string, len(s))
	for i, a := range s {
		ids[i] = a.ID
	}
	return ids
}

func TestGetAlbumSummaries(t *testing.T) {
	lib, ctx := seedAlbumLibrary(t)

	all, err := lib.GetAlbumSummaries(ctx, 50, 0, &music.AlbumFilter{})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*music.AlbumSummary{}
	for _, s := range all {
		byID[s.ID] = s
	}
	if len(all) != 3 {
		t.Fatalf("want 3 albums, got %v", summaryIDs(all))
	}

	z := byID["al-zeta"]
	if z.TrackCount != 3 || z.Duration != 600 || z.Year != 2001 || z.Artists != "Alpha" {
		t.Errorf("zeta summary wrong: %+v", z)
	}
	if z.CoverTrackID != "t-z-1-1" {
		t.Errorf("cover should be disc 1 track 1, got %q", z.CoverTrackID)
	}
	if b := byID["al-beta"]; b.Year != 1999 {
		t.Errorf("beta should fall back to track year 1999, got %d", b.Year)
	}
	if e := byID["al-empty"]; e.TrackCount != 0 || e.CoverTrackID != "" || e.Duration != 0 {
		t.Errorf("empty album summary wrong: %+v", e)
	}

	// Default sort: artist, then year, then title → Alpha/Empty(0), Alpha/Zeta(2001), Bravo/Beta.
	if got, want := summaryIDs(all), []string{"al-empty", "al-zeta", "al-beta"}; !equal(got, want) {
		t.Errorf("artist sort: got %v want %v", got, want)
	}

	cases := []struct {
		name   string
		filter music.AlbumFilter
		want   []string
	}{
		{"title sort", music.AlbumFilter{Sort: music.AlbumSortTitle}, []string{"al-beta", "al-empty", "al-zeta"}},
		{"year sort", music.AlbumFilter{Sort: music.AlbumSortYear}, []string{"al-zeta", "al-beta", "al-empty"}},
		{"recent sort", music.AlbumFilter{Sort: music.AlbumSortRecent}, []string{"al-beta", "al-zeta", "al-empty"}},
		{"search by title", music.AlbumFilter{TextSearch: "zet"}, []string{"al-zeta"}},
		{"search by album artist", music.AlbumFilter{TextSearch: "brav"}, []string{"al-beta"}},
		{"genre", music.AlbumFilter{Genre: "Jazz"}, []string{"al-beta"}},
		{"search and genre", music.AlbumFilter{TextSearch: "alpha", Genre: "Rock"}, []string{"al-zeta"}},
	}
	for _, tc := range cases {
		got, err := lib.GetAlbumSummaries(ctx, 50, 0, &tc.filter)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !equal(summaryIDs(got), tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, summaryIDs(got), tc.want)
		}
		count, err := lib.GetAlbumSummariesCount(ctx, &tc.filter)
		if err != nil {
			t.Fatalf("%s count: %v", tc.name, err)
		}
		if count != len(tc.want) {
			t.Errorf("%s: count %d want %d", tc.name, count, len(tc.want))
		}
	}

	// A genre filter selects albums, not tracks: a mixed-genre album keeps its full aggregates.
	mixed, err := lib.GetAlbumSummaries(ctx, 50, 0, &music.AlbumFilter{Genre: "Electronic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mixed) != 1 || mixed[0].ID != "al-zeta" || mixed[0].TrackCount != 3 || mixed[0].Duration != 600 {
		t.Errorf("genre filter should return zeta with all 3 tracks / 600s, got %+v", mixed)
	}

	// Pagination
	page2, err := lib.GetAlbumSummaries(ctx, 2, 2, &music.AlbumFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !equal(summaryIDs(page2), []string{"al-beta"}) {
		t.Errorf("page 2: got %v", summaryIDs(page2))
	}
}

func TestGetAlbumTracksOrdersByDiscAndTrack(t *testing.T) {
	lib, ctx := seedAlbumLibrary(t)
	tracks, err := lib.GetAlbumTracks(ctx, "al-zeta")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tr := range tracks {
		got = append(got, tr.ID)
	}
	if want := []string{"t-z-1-1", "t-z-1-2", "t-z-2-1"}; !equal(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
