package metadata

import (
	"context"

	"github.com/contre95/soulsolid/src/features/tagging"
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

func (p *DiscogsProvider) SearchTracks(ctx context.Context, params tagging.SearchParams) ([]*music.Track, error) {
	// Mock implementation - simulate search results from Discogs API
	// In a real implementation, this would query the Discogs API with the search parameters

	// If we have search parameters, return relevant results
	if params.Title != "" || params.AlbumArtist != "" {
		tracks := []*music.Track{
			{
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
			},
			{
				Title: "Whole Lotta Love",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Led Zeppelin II",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "Led Zeppelin"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1969,
					Genre:       "Rock",
					TrackNumber: 1,
					Composer:    "Jimmy Page, Robert Plant, John Paul Jones",
				},
				ISRC: "USAT29900470",
			},
			{
				Title: "A Day in the Life",
				Artists: []music.ArtistRole{
					{Artist: &music.Artist{Name: "The Beatles"}, Role: "main"},
				},
				Album: &music.Album{
					Title: "Sgt. Pepper's Lonely Hearts Club Band",
					Artists: []music.ArtistRole{
						{Artist: &music.Artist{Name: "The Beatles"}, Role: "main"},
					},
				},
				Metadata: music.Metadata{
					Year:        1967,
					Genre:       "Rock",
					TrackNumber: 13,
					Composer:    "John Lennon, Paul McCartney",
				},
				ISRC: "GBAYE0601698",
			},
		}
		return tracks, nil
	}

	// Default fallback results
	tracks := []*music.Track{
		{
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
		},
	}

	return tracks, nil
}

func (p *DiscogsProvider) FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error) {
	// TODO: Implement full Discogs API integration
	// For now, return realistic placeholder data that demonstrates the functionality
	tracks, err := p.SearchTracks(ctx, tagging.SearchParams{})
	if err != nil || len(tracks) == 0 {
		return nil, err
	}
	return tracks[0], nil
}

func (p *DiscogsProvider) Name() string    { return "discogs" }
func (p *DiscogsProvider) IsEnabled() bool { return p.enabled }
