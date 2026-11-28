package providers

import "testing"

func TestNormalizeMatch(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"OK Computer", "okcomputer"},
		{"ok  computer!", "okcomputer"},
		{"Sgt. Pepper's Lonely Hearts Club Band", "sgtpepperslonelyheartsclubband"},
		{"Björk", "björk"},
		{"", ""},
		{"...", ""},
	}
	for _, tt := range tests {
		if got := normalizeMatch(tt.in); got != tt.want {
			t.Errorf("normalizeMatch(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStripQuotes(t *testing.T) {
	if got := stripQuotes(` He said "hi" \o/ `); got != `He said hi o/` {
		t.Errorf("stripQuotes = %q", got)
	}
}

func TestPickMBRelease(t *testing.T) {
	rel := func(id, status string, tracks int) mbReleaseResult {
		return mbReleaseResult{ID: id, Status: status, TrackCount: tracks}
	}
	tests := []struct {
		name       string
		candidates []mbReleaseResult
		wantID     string
		wantOK     bool
	}{
		{"none", nil, "", false},
		{"single", []mbReleaseResult{rel("a", "Official", 12)}, "a", true},
		{"most common count wins", []mbReleaseResult{rel("a", "Official", 14), rel("b", "Official", 12), rel("c", "Official", 12)}, "b", true},
		{"tie goes to the smaller count", []mbReleaseResult{rel("a", "Official", 14), rel("b", "Official", 12)}, "b", true},
		{"official beats promotional", []mbReleaseResult{rel("a", "Promotion", 3), rel("b", "Promotion", 3), rel("c", "Official", 10)}, "c", true},
		{"falls back to any status", []mbReleaseResult{rel("a", "Bootleg", 9)}, "a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pickMBRelease(tt.candidates)
			if ok != tt.wantOK || got.ID != tt.wantID {
				t.Errorf("pickMBRelease() = (%q, %v), want (%q, %v)", got.ID, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}
