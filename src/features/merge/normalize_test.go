package merge

import (
	"sync"
	"testing"
)

// TestNormalizeKeyConcurrent guards against sharing a stateful transformer: run -race.
func TestNormalizeKeyConcurrent(t *testing.T) {
	inputs := []string{"Un Verano Sin Tí", "hip-hop", "Bad Bunny", "Café Tacvba", "AC/DC"}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, in := range inputs {
				_ = normalizeKey(in)
			}
		}()
	}
	wg.Wait()
}

func TestNormalizeKeyMatches(t *testing.T) {
	groups := [][]string{
		{"un verano sin ti", "Un Verano Sin Ti", "Un Verano Sin Tí", "un-verano-sin-ti"},
		{"hip-hop", "Hip Hop", "HipHop", "HIP HOP"},
		{"Bad Bunny", "bad bunny", "BAD  BUNNY"},
	}
	for _, g := range groups {
		want := normalizeKey(g[0])
		for _, v := range g[1:] {
			if got := normalizeKey(v); got != want {
				t.Errorf("normalizeKey(%q)=%q, want %q (from %q)", v, got, want, g[0])
			}
		}
	}
}

func TestNormalizeKeyDistinct(t *testing.T) {
	if normalizeKey("Rock") == normalizeKey("Rock and Roll") {
		t.Error("distinct names should not collapse to the same key")
	}
}

func TestSmartCanonical(t *testing.T) {
	if got := smartCanonical([]string{"un verano sin ti", "Un Verano Sin Ti"}); got != "Un Verano Sin Ti" {
		t.Errorf("smartCanonical picked %q, want proper-case variant", got)
	}
	if got := smartCanonical([]string{"HIP HOP", "Hip Hop", "hiphop"}); got != "Hip Hop" {
		t.Errorf("smartCanonical picked %q, want %q", got, "Hip Hop")
	}
}
