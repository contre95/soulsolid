package database

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/contre95/soulsolid/src/music"
)

// seedAlbum adds an album with the given number of local tracks and attributes.
func seedAlbum(t *testing.T, db *SqliteLibrary, artist *music.Artist, id, title string, tracks int, attrs map[string]string) {
	t.Helper()
	ctx := context.Background()
	album := &music.Album{
		ID: id, Title: title, Attributes: attrs,
		Artists: []music.ArtistRole{{Artist: artist, Role: "main"}},
	}
	if err := db.AddAlbum(ctx, album); err != nil {
		t.Fatalf("AddAlbum(%s): %v", title, err)
	}
	for i := 1; i <= tracks; i++ {
		track := &music.Track{
			ID:      fmt.Sprintf("%s-t%d", id, i),
			Path:    fmt.Sprintf("/music/%s/%d.flac", id, i),
			Title:   fmt.Sprintf("%s track %d", title, i),
			Artists: []music.ArtistRole{{Artist: artist, Role: "main"}},
			Album:   album,
			Metadata: music.Metadata{
				TrackNumber: i,
				DiscNumber:  1,
			},
		}
		if err := db.AddTrack(ctx, track); err != nil {
			t.Fatalf("AddTrack(%s): %v", track.Title, err)
		}
	}
}

func expected(n int, source string) map[string]string {
	return map[string]string{
		music.AlbumAttrExpectedTracks:        fmt.Sprint(n),
		music.AlbumAttrCompletenessSource:    source,
		music.AlbumAttrCompletenessCheckedAt: "2026-01-02T03:04:05Z",
	}
}

func TestIncompleteAlbums(t *testing.T) {
	ctx := context.Background()
	db, err := NewSqliteLibrary(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("NewSqliteLibrary: %v", err)
	}
	artist := &music.Artist{ID: "artist-1", Name: "Radiohead"}
	if err := db.AddArtist(ctx, artist); err != nil {
		t.Fatalf("AddArtist: %v", err)
	}

	seedAlbum(t, db, artist, "short", "Missing Two", 3, expected(5, "deezer"))
	seedAlbum(t, db, artist, "shorter", "Missing Three", 1, expected(4, "musicbrainz"))
	seedAlbum(t, db, artist, "full", "Complete", 2, expected(2, "deezer"))
	seedAlbum(t, db, artist, "extra", "Bonus Tracks", 4, expected(3, "deezer"))
	seedAlbum(t, db, artist, "unchecked", "Never Checked", 3, nil)
	seedAlbum(t, db, artist, "empty", "No Tracks Left", 0, expected(6, "manual"))

	count, err := db.GetIncompleteAlbumsCount(ctx)
	if err != nil {
		t.Fatalf("GetIncompleteAlbumsCount: %v", err)
	}
	if count != 3 {
		t.Errorf("incomplete count = %d, want 3", count)
	}

	rows, err := db.GetIncompleteAlbums(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetIncompleteAlbums: %v", err)
	}
	// Most missing tracks first.
	wantOrder := []string{"empty", "shorter", "short"}
	if len(rows) != len(wantOrder) {
		t.Fatalf("got %d rows, want %d: %+v", len(rows), len(wantOrder), rows)
	}
	for i, id := range wantOrder {
		if rows[i].ID != id {
			t.Errorf("row %d = %s, want %s", i, rows[i].ID, id)
		}
	}

	short := rows[2]
	if short.Title != "Missing Two" || short.Artists != "Radiohead" || short.LocalCount != 3 || short.ExpectedCount != 5 ||
		short.Missing() != 2 || short.Source != "deezer" || short.CheckedAt == "" || short.CoverTrackID != "short-t1" {
		t.Errorf("unexpected row: %+v", short)
	}
	if rows[0].CoverTrackID != "" || rows[0].LocalCount != 0 || rows[0].Source != music.CompletenessSourceManual {
		t.Errorf("unexpected row for the album without tracks: %+v", rows[0])
	}

	page, err := db.GetIncompleteAlbums(ctx, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != "shorter" {
		t.Errorf("second page = %+v, %v", page, err)
	}
}

func TestSetAlbumAttributesUpserts(t *testing.T) {
	ctx := context.Background()
	db, err := NewSqliteLibrary(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("NewSqliteLibrary: %v", err)
	}
	artist := &music.Artist{ID: "artist-1", Name: "Radiohead"}
	if err := db.AddArtist(ctx, artist); err != nil {
		t.Fatalf("AddArtist: %v", err)
	}
	seedAlbum(t, db, artist, "a1", "Kid A", 2, map[string]string{"keep": "me", music.AlbumAttrExpectedTracks: "9"})

	if err := db.SetAlbumAttributes(ctx, "a1", map[string]string{
		music.AlbumAttrExpectedTracks:     "2",
		music.AlbumAttrCompletenessSource: "deezer",
	}); err != nil {
		t.Fatalf("SetAlbumAttributes: %v", err)
	}

	album, err := db.GetAlbum(ctx, "a1")
	if err != nil || album == nil {
		t.Fatalf("GetAlbum: %v, %v", album, err)
	}
	if album.Attributes["keep"] != "me" {
		t.Errorf("unrelated attributes must survive, got %v", album.Attributes)
	}
	if album.Attributes[music.AlbumAttrExpectedTracks] != "2" || album.Attributes[music.AlbumAttrCompletenessSource] != "deezer" {
		t.Errorf("attributes not updated: %v", album.Attributes)
	}

	// The album was incomplete (2 of 9) and is complete now.
	if count, err := db.GetIncompleteAlbumsCount(ctx); err != nil || count != 0 {
		t.Errorf("incomplete count = %d, %v, want 0", count, err)
	}
}
