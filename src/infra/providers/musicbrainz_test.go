package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/contre95/soulsolid/src/features/metadata"
)

func TestSearchTracks_JoinsTermsWithAND(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query().Get("query")
		w.Write([]byte(`{"recordings": []}`))
	}))
	defer server.Close()
	provider := NewMusicBrainzProvider(true)
	provider.baseURL = server.URL

	if _, err := provider.SearchTracks(context.Background(), metadata.SearchParams{
		Title:       "Round Trip",
		AlbumArtist: "Album Artist",
		Album:       "Test Album",
		Year:        2001,
		AcoustID:    "00000000-0000-0000-0000-000000000000",
	}); err != nil {
		t.Fatalf("SearchTracks returned error: %v", err)
	}
	want := `recording:"Round Trip" AND artist:"Album Artist" AND release:"Test Album" AND date:2001`
	if query != want {
		t.Errorf("expected query %q, got %q", want, query)
	}
}
