package metadata

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// DiscogsProvider implements MetadataProvider for Discogs
type DiscogsProvider struct {
	enabled bool
	apiKey  string
}

// NewDiscogsProvider creates a new Discogs provider
func NewDiscogsProvider(enabled bool, apiKey string) *DiscogsProvider {
	return &DiscogsProvider{enabled: enabled, apiKey: apiKey}
}

func (p *DiscogsProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Discogs API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	return &music.Track{
		Title: "Stairway to Heaven",
		Artists: []music.ArtistRole{
			{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
		},
		Album: &music.Album{
			Title: "Led Zeppelin IV",
			Artists: []music.ArtistRole{
				{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
			},
		},
		Metadata: music.Metadata{
			Year:        1971,
			Genre:       "Rock",
			TrackNumber: 4,
			Composer:    "Jimmy Page, Robert Plant",
		},
		ISRC: "USAT21300959",
	}, nil
}

func (p *DiscogsProvider) Name() string    { return "discogs" }
func (p *DiscogsProvider) IsEnabled() bool { return p.enabled }
