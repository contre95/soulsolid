package database

import (
	"context"
	"testing"
)

func newTestLibrary(t *testing.T, paths ...string) *SqliteLibrary {
	t.Helper()

	lib, err := NewSqliteLibrary(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory library: %v", err)
	}
	t.Cleanup(func() { _ = lib.db.Close() })

	for i, path := range paths {
		if _, err := lib.db.Exec(
			`INSERT INTO tracks (id, path, title) VALUES (?, ?, ?)`,
			string(rune('a'+i)), path, "track",
		); err != nil {
			t.Fatalf("failed to insert track %q: %v", path, err)
		}
	}
	return lib
}

func TestInspectLibraryRoot(t *testing.T) {
	tests := []struct {
		name        string
		root        string
		paths       []string
		wantTotal   int
		wantOutside int
	}{
		{
			name:      "empty database never reports a mismatch",
			root:      "/music",
			wantTotal: 0,
		},
		{
			name:      "all tracks under the configured root",
			root:      "/music",
			paths:     []string{"/music/Artist/Album/01 A.flac", "/music/Other/02 B.mp3"},
			wantTotal: 2,
		},
		{
			name:      "trailing separator on the root is tolerated",
			root:      "/music/",
			paths:     []string{"/music/Artist/Album/01 A.flac"},
			wantTotal: 1,
		},
		{
			name:      "relative root matches relative stored paths",
			root:      "./music",
			paths:     []string{"music/Artist/Album/01 A.flac"},
			wantTotal: 1,
		},
		{
			name:        "sibling directory sharing a prefix is not a match",
			root:        "/music",
			paths:       []string{"/musicvideos/Artist/01 A.flac"},
			wantTotal:   1,
			wantOutside: 1,
		},
		{
			name:        "root changed since import",
			root:        "/srv/music",
			paths:       []string{"/old/music/A.flac", "/old/music/B.flac", "/srv/music/C.flac"},
			wantTotal:   3,
			wantOutside: 2,
		},
		{
			name:        "absolute root does not match relative stored paths",
			root:        "/music",
			paths:       []string{"music/Artist/01 A.flac"},
			wantTotal:   1,
			wantOutside: 1,
		},
		{
			name:      "non-ascii root offsets are counted in runes",
			root:      "/músïc",
			paths:     []string{"/músïc/Artist/01 A.flac"},
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lib := newTestLibrary(t, tt.paths...)

			report, err := lib.InspectLibraryRoot(context.Background(), tt.root)
			if err != nil {
				t.Fatalf("InspectLibraryRoot returned an error: %v", err)
			}

			if report.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", report.Total, tt.wantTotal)
			}
			if report.Outside != tt.wantOutside {
				t.Errorf("Outside = %d, want %d", report.Outside, tt.wantOutside)
			}
			if got, want := report.Mismatch(), tt.wantOutside > 0; got != want {
				t.Errorf("Mismatch() = %v, want %v", got, want)
			}
			if tt.wantOutside > 0 && len(report.Samples) == 0 {
				t.Error("Samples is empty, want at least one offending path for diagnostics")
			}
			if tt.wantOutside == 0 && len(report.Samples) != 0 {
				t.Errorf("Samples = %v, want none when there is no mismatch", report.Samples)
			}
		})
	}
}

func TestInspectLibraryRootSamplesAreCapped(t *testing.T) {
	paths := []string{"/old/A.flac", "/old/B.flac", "/old/C.flac", "/old/D.flac", "/old/E.flac"}
	lib := newTestLibrary(t, paths...)

	report, err := lib.InspectLibraryRoot(context.Background(), "/music")
	if err != nil {
		t.Fatalf("InspectLibraryRoot returned an error: %v", err)
	}

	if report.Outside != len(paths) {
		t.Errorf("Outside = %d, want %d", report.Outside, len(paths))
	}
	if len(report.Samples) > 3 {
		t.Errorf("Samples has %d entries, want at most 3", len(report.Samples))
	}
}
