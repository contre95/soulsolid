package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/contre95/soulsolid/src/music"
)

func TestResolveImportPath(t *testing.T) {
	base := t.TempDir()
	downloads := filepath.Join(base, "downloads")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{filepath.Join(downloads, "album"), outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(downloads, "escape")); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, path string
		ok         bool
	}{
		{"empty imports the download path", "", true},
		{"relative inside", "album", true},
		{"absolute inside", filepath.Join(downloads, "album"), true},
		{"absolute outside", outside, false},
		{"dot-dot escape", "../outside", false},
		{"symlink escape", "escape", false},
		{"missing", "nope", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveImportPath(downloads, tc.path)
			if (err == nil) != tc.ok {
				t.Errorf("resolveImportPath(%q): ok=%v, err=%v", tc.path, tc.ok, err)
			}
		})
	}
}

func TestTagFormData(t *testing.T) {
	track := &music.Track{
		Title:     "Song",
		HasLyrics: true,
		Metadata:  music.Metadata{Year: 1999, Genre: "Rock", TrackNumber: 3, Composer: "Someone", Lyrics: "la la"},
	}

	// Numbers sent for string parameters must be applied, not dropped.
	f := tagFormData(track, map[string]any{"year": float64(2004), "track_number": float64(7)})
	if f["year"] != "2004" || f["track_number"] != "7" {
		t.Errorf("numeric args not applied: year=%q track_number=%q", f["year"], f["track_number"])
	}
	if f["title"] != "Song" || f["genre"] != "Rock" {
		t.Errorf("omitted fields must keep current values: %v", f)
	}

	// Empty strings clear clearable fields but never the title.
	f = tagFormData(track, map[string]any{"genre": "", "lyrics": "", "composer": "", "title": ""})
	if f["genre"] != "" || f["lyrics"] != "" || f["composer"] != "" {
		t.Errorf("expected genre, lyrics and composer cleared: %v", f)
	}
	if f["title"] != "Song" {
		t.Errorf("title must not be cleared, got %q", f["title"])
	}
}

func TestTruncation(t *testing.T) {
	big := strings.Repeat("0123456789abcdef\n", maxTextResult/8)
	out := truncateText(big)
	if len(out) > maxTextResult+100 || !strings.Contains(out, "[truncated:") {
		t.Errorf("expected truncated output, got %d bytes", len(out))
	}
	if truncateText("small") != "small" {
		t.Error("small text must be returned unchanged")
	}

	logs := "a\nb\nc\nd\n"
	if got := tailLines(logs, 2); !strings.HasSuffix(got, "c\nd") || !strings.Contains(got, "last 2 of 4") {
		t.Errorf("tailLines: got %q", got)
	}
	if got := tailLines(logs, 10); got != logs {
		t.Errorf("tailLines with fewer lines than n: got %q", got)
	}

	if got := limitArg(map[string]any{"limit": float64(5000)}, "limit", 20); got != maxListLimit {
		t.Errorf("limitArg must clamp to %d, got %d", maxListLimit, got)
	}
}
